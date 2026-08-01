package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var defaultAllowedIfaces = []string{"lo", "br+", "tun+", "tap+", "wg+"}

const ifaceCacheTTL = 15 * time.Second

func allowedIfacePatterns(env string) ([]string, bool) {
	fields := strings.Fields(strings.ReplaceAll(env, ",", " "))
	if len(fields) == 0 {
		return defaultAllowedIfaces, false
	}

	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f == "+" || !validIfacePattern(f) {
			return defaultAllowedIfaces, false
		}
		out = append(out, f)
	}
	return out, true
}

func validIfacePattern(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '+', r == '-':
		default:
			return false
		}
	}
	return true
}

func ifaceAllowed(name string, patterns []string) bool {
	for _, p := range patterns {
		if prefix, wildcard := strings.CutSuffix(p, "+"); wildcard {
			if strings.HasPrefix(name, prefix) {
				return true
			}
		} else if name == p {
			return true
		}
	}
	return false
}

func privateAddr(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

type ifaceIndex map[string][]string

func buildIfaceIndex() (ifaceIndex, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	idx := make(ifaceIndex)
	for _, ifi := range ifaces {
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			key := ip.String()
			idx[key] = append(idx[key], ifi.Name)
		}
	}
	return idx, nil
}

func (idx ifaceIndex) allows(ip net.IP, patterns []string) (allowed, known bool) {
	if ip == nil {
		return false, true
	}

	names, ok := idx[ip.String()]
	if !ok || len(names) == 0 {
		return false, false
	}

	for _, n := range names {
		if !ifaceAllowed(n, patterns) {
			return false, true
		}
	}
	return true, true
}

func (idx ifaceIndex) inventory(patterns []string) string {
	if len(idx) == 0 {
		return "(no interfaces visible)"
	}

	keys := make([]string, 0, len(idx))
	for k := range idx {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		names := idx[k]
		verdict := "ok"
		for _, n := range names {
			if !ifaceAllowed(n, patterns) {
				verdict = "refused"
				break
			}
		}
		fmt.Fprintf(&b, " %s[%s %s]", k, strings.Join(names, ","), verdict)
	}
	return strings.TrimSpace(b.String())
}

type lanGuard struct {
	patterns []string

	mu      sync.Mutex
	idx     ifaceIndex
	fetched time.Time
	warned  map[string]bool
	build   func() (ifaceIndex, error)
}

func newLANGuard() *lanGuard {
	env := os.Getenv("IDEFIX_ALLOW_IFACES")
	patterns, used := allowedIfacePatterns(env)

	if env != "" && !used {
		log.Printf("ignoring malformed IDEFIX_ALLOW_IFACES (%q), using defaults", env)
	}

	return &lanGuard{
		patterns: patterns,
		build:    buildIfaceIndex,
	}
}

func (g *lanGuard) index(refresh bool) (ifaceIndex, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !refresh && g.idx != nil && time.Since(g.fetched) < ifaceCacheTTL {
		return g.idx, nil
	}

	idx, err := g.build()
	if err != nil {
		return nil, err
	}
	g.idx, g.fetched = idx, time.Now()
	return idx, nil
}

func (g *lanGuard) permits(local net.Addr) bool {
	var ip net.IP
	switch v := local.(type) {
	case *net.TCPAddr:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	default:
		host, _, err := net.SplitHostPort(local.String())
		if err != nil {
			return false
		}
		ip = net.ParseIP(host)
	}
	if ip == nil {
		return false
	}

	idx, err := g.index(false)
	if err != nil {
		return privateAddr(ip)
	}

	allowed, known := idx.allows(ip, g.patterns)
	if known {
		return allowed
	}

	if idx, err = g.index(true); err != nil {
		return privateAddr(ip)
	}
	if allowed, known = idx.allows(ip, g.patterns); known {
		return allowed
	}

	g.warnUnplaceable(ip)
	return privateAddr(ip)
}

func (g *lanGuard) warnUnplaceable(ip net.IP) {
	key := ip.String()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.warned == nil {
		g.warned = make(map[string]bool)
	}
	if g.warned[key] || len(g.warned) >= 32 {
		return
	}
	g.warned[key] = true

	log.Printf("local address %s belongs to no interface we can see; "+
		"falling back to allowing private addresses only", key)
}

type guardedListener struct {
	net.Listener
	guard *lanGuard
}

func (l *guardedListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		if l.guard.permits(c.LocalAddr()) {
			return c, nil
		}

		log.Printf("refused %s: connection arrived on non-LAN address %s", c.RemoteAddr(), c.LocalAddr())
		c.Close()
	}
}
