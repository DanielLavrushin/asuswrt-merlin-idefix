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

// This port hands out a root shell, so it must never answer a connection that
// arrived from the Internet. scripts/firewall.sh is the first line of defence;
// this is the second, covering the window where those rules are not in force —
// a firewall restart that rebuilt the chains, another addon flushing INPUT, a
// hand-edited rule.
//
// A connection is judged by the local address it landed on rather than by its
// source address: that address belongs to a specific interface, and only the
// loopback, the LAN bridges and the router's own VPN tunnels are acceptable.
// Source addresses prove nothing here — a packet from the WAN can carry any
// source it likes.
//
// Patterns use the iptables grammar: a trailing '+' is a wildcard, anything
// else is an exact interface name. The firewall rules are written from the same
// value, and the two must agree — a pattern that does not open an interface in
// iptables must not open it here either.
var defaultAllowedIfaces = []string{"lo", "br+", "tun+", "tap+", "wg+"}

// ifaceCacheTTL bounds how long a connection can be judged against a stale
// picture of the router's interfaces. Tunnels come and go as VPNs connect.
const ifaceCacheTTL = 15 * time.Second

// allowedIfacePatterns mirrors IDEFIX_ALLOW_IFACES from firewall.sh so one
// setting covers both layers. Spaces and commas both separate entries, matching
// what the shell side accepts.
func allowedIfacePatterns(env string) []string {
	env = strings.TrimSpace(env)
	if env == "" {
		return defaultAllowedIfaces
	}

	var out []string
	for _, f := range strings.Fields(strings.ReplaceAll(env, ",", " ")) {
		// A bare "+" would match every interface, the WAN included.
		if f != "" && f != "+" {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return defaultAllowedIfaces
	}
	return out
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

// privateAddr reports whether an address can only have come from a private
// network. Used as a fallback when the interface list cannot be read at all:
// it still refuses a publicly routable WAN address, which is the case that
// matters, without making the terminal unreachable on firmware where
// enumerating interfaces fails.
func privateAddr(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

// ifaceIndex maps an interface address to the names of every interface holding
// it. Link-local addresses are not unique across interfaces, hence the slice.
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
			// One unreadable interface must not blind us to the rest; an
			// address we then fail to find is refused, not allowed.
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

// allows reports whether every interface carrying ip is allowed. An address we
// cannot place is not allowed: unknown means refused, never permitted.
func (idx ifaceIndex) allows(ip net.IP, patterns []string) (allowed, known bool) {
	if ip == nil {
		return false, true
	}

	// A v4 connection on a dual-stack listener reports its local address as
	// 4-in-6; String() normalises both forms to the same dotted quad.
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

// inventory renders what the guard can see, as "addr[iface ok|refused]". Logged
// once at startup: when a client is refused, the first question is always what
// the router reported, and that cannot be reconstructed after the fact.
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

// lanGuard decides, per connection, whether the interface it arrived on is one
// the terminal may serve.
type lanGuard struct {
	patterns []string

	mu      sync.Mutex
	idx     ifaceIndex
	fetched time.Time
	warned  map[string]bool
	build   func() (ifaceIndex, error)
}

func newLANGuard() *lanGuard {
	return &lanGuard{
		patterns: allowedIfacePatterns(os.Getenv("IDEFIX_ALLOW_IFACES")),
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
		// Interfaces unreadable: fall back to refusing anything publicly
		// routable rather than refusing everything.
		return privateAddr(ip)
	}

	allowed, known := idx.allows(ip, g.patterns)
	if known {
		return allowed
	}

	// A tunnel that came up since the cache was built looks unknown. Rebuild
	// once before deciding.
	if idx, err = g.index(true); err != nil {
		return privateAddr(ip)
	}
	if allowed, known = idx.allows(ip, g.patterns); known {
		return allowed
	}

	// The address is on no interface we can see. That means our picture of the
	// router is incomplete, not that the address is suspicious — refusing here
	// takes the terminal down for LAN clients, which is what happened on
	// firmware where the LAN bridge does not enumerate. Fall back to the
	// address test: a WAN address is publicly routable and still refused.
	g.warnUnplaceable(ip)
	return privateAddr(ip)
}

// warnUnplaceable reports an address we could not attribute to an interface,
// once per address, so a misread of the router shows up in the system log
// instead of silently widening what the guard accepts.
func (g *lanGuard) warnUnplaceable(ip net.IP) {
	key := ip.String()

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.warned == nil {
		g.warned = make(map[string]bool)
	}
	// Bounded: an attacker must not be able to grow this by connecting.
	if g.warned[key] || len(g.warned) >= 32 {
		return
	}
	g.warned[key] = true

	log.Printf("local address %s belongs to no interface we can see; "+
		"falling back to allowing private addresses only", key)
}

// guardedListener drops refused connections inside Accept, so they never reach
// the HTTP server and a refusal is not mistaken for a listener failure.
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
