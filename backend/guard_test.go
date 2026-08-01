package main

import (
	"errors"
	"fmt"
	"net"
	"testing"
)

func testGuard(idx ifaceIndex, err error, patterns []string) *lanGuard {
	g := &lanGuard{patterns: patterns}
	g.build = func() (ifaceIndex, error) { return idx, err }
	return g
}

// A router with a LAN bridge, a guest bridge, an OpenVPN server, and a WAN.
func routerIndex() ifaceIndex {
	return ifaceIndex{
		"127.0.0.1":     {"lo"},
		"::1":           {"lo"},
		"192.168.1.1":   {"br0"},
		"192.168.101.1": {"br1"},
		"10.8.0.1":      {"tun21"},
		"10.6.0.1":      {"wgs1"},
		"203.0.113.7":   {"eth0"}, // WAN, public
		"100.64.3.9":    {"ppp0"}, // WAN behind carrier NAT
		"fd00::1":       {"br0"},
		"2001:db8::1":   {"eth0"}, // WAN, routable IPv6
	}
}

func TestGuardRefusesWANAddresses(t *testing.T) {
	g := testGuard(routerIndex(), nil, defaultAllowedIfaces)

	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"192.168.1.1", true},
		{"192.168.101.1", true},
		{"10.8.0.1", true},
		{"10.6.0.1", true},
		{"fd00::1", true},

		{"203.0.113.7", false}, // the address the scanners in issue #12 hit
		{"100.64.3.9", false},  // private-looking, but it is the WAN
		{"2001:db8::1", false}, // IPv6 WAN, which the v4 rules alone would miss
	}

	for _, c := range cases {
		got := g.permits(&net.TCPAddr{IP: net.ParseIP(c.ip), Port: 8787})
		if got != c.want {
			t.Errorf("permits(local=%s) = %v, want %v", c.ip, got, c.want)
		}
	}
}

// A dual-stack listener reports an IPv4 peer's local address in 4-in-6 form.
func TestGuardHandlesIPv4MappedAddresses(t *testing.T) {
	g := testGuard(routerIndex(), nil, defaultAllowedIfaces)

	if !g.permits(&net.TCPAddr{IP: net.ParseIP("::ffff:192.168.1.1"), Port: 8787}) {
		t.Error("4-in-6 LAN address refused")
	}
	if g.permits(&net.TCPAddr{IP: net.ParseIP("::ffff:203.0.113.7"), Port: 8787}) {
		t.Error("4-in-6 WAN address permitted")
	}
}

// An address on no interface we can see means our picture of the router is
// incomplete. Refusing outright takes the terminal down for LAN clients on
// firmware whose bridge does not enumerate, so the decision falls back to the
// address itself — which still refuses anything publicly routable.
func TestGuardFallsBackForUnplaceableAddresses(t *testing.T) {
	g := testGuard(routerIndex(), nil, defaultAllowedIfaces)

	if g.permits(&net.TCPAddr{IP: net.ParseIP("198.51.100.5"), Port: 8787}) {
		t.Error("unplaceable public address was permitted")
	}
	if g.permits(&net.TCPAddr{IP: net.ParseIP("2001:db8:dead::1"), Port: 8787}) {
		t.Error("unplaceable public IPv6 address was permitted")
	}
	if !g.permits(&net.TCPAddr{IP: net.ParseIP("192.168.50.1"), Port: 8787}) {
		t.Error("unplaceable private address was refused, which strands LAN clients")
	}
	if g.permits(&net.TCPAddr{IP: nil, Port: 8787}) {
		t.Error("nil address was permitted")
	}
}

// The exact shape seen on the router: the LAN bridge is missing from the index
// entirely, so every LAN client was being turned away.
func TestGuardServesLANWhenBridgeDoesNotEnumerate(t *testing.T) {
	// Only the loopback enumerated; br0 and its 192.168.1.1 never appeared.
	g := testGuard(ifaceIndex{"127.0.0.1": {"lo"}}, nil, defaultAllowedIfaces)

	if !g.permits(&net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8787}) {
		t.Error("LAN address refused because the bridge did not enumerate")
	}
	if g.permits(&net.TCPAddr{IP: net.ParseIP("203.0.113.7"), Port: 8787}) {
		t.Error("WAN address permitted once enumeration was incomplete")
	}
}

func TestGuardWarnsOncePerUnplaceableAddress(t *testing.T) {
	g := testGuard(ifaceIndex{"127.0.0.1": {"lo"}}, nil, defaultAllowedIfaces)

	for i := 0; i < 5; i++ {
		g.permits(&net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8787})
	}
	if len(g.warned) != 1 {
		t.Errorf("warned about %d addresses, want 1", len(g.warned))
	}

	// Bounded, so connections cannot grow it without limit.
	for i := 0; i < 100; i++ {
		g.permits(&net.TCPAddr{IP: net.ParseIP(fmt.Sprintf("10.0.%d.1", i)), Port: 8787})
	}
	if len(g.warned) > 32 {
		t.Errorf("warned map grew to %d entries, want it capped at 32", len(g.warned))
	}
}

// A link-local address can sit on several interfaces at once; if any of them is
// the WAN the connection has to go.
func TestGuardRefusesAddressSharedWithWAN(t *testing.T) {
	idx := ifaceIndex{"fe80::1": {"br0", "eth0"}}
	g := testGuard(idx, nil, defaultAllowedIfaces)

	if g.permits(&net.TCPAddr{IP: net.ParseIP("fe80::1"), Port: 8787}) {
		t.Error("address shared with a WAN interface was permitted")
	}
}

// When interfaces cannot be enumerated at all we still refuse anything
// publicly routable, rather than either refusing everything or trusting it.
func TestGuardFallsBackToPrivateAddressesOnly(t *testing.T) {
	g := testGuard(nil, errors.New("no netlink"), defaultAllowedIfaces)

	if !g.permits(&net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8787}) {
		t.Error("private address refused under the fallback")
	}
	if g.permits(&net.TCPAddr{IP: net.ParseIP("203.0.113.7"), Port: 8787}) {
		t.Error("public address permitted under the fallback")
	}
	if g.permits(&net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 8787}) {
		t.Error("public IPv6 address permitted under the fallback")
	}
}

// A tunnel that comes up after the cache was built must not be refused
// forever — one miss triggers a rebuild.
func TestGuardRefreshesForNewInterfaces(t *testing.T) {
	g := &lanGuard{patterns: defaultAllowedIfaces}
	calls := 0
	g.build = func() (ifaceIndex, error) {
		calls++
		if calls == 1 {
			return ifaceIndex{"192.168.1.1": {"br0"}}, nil
		}
		return ifaceIndex{"192.168.1.1": {"br0"}, "10.8.0.1": {"tun21"}}, nil
	}

	if !g.permits(&net.TCPAddr{IP: net.ParseIP("192.168.1.1")}) {
		t.Fatal("LAN address refused")
	}
	if !g.permits(&net.TCPAddr{IP: net.ParseIP("10.8.0.1")}) {
		t.Error("address on a newly created tunnel refused after refresh")
	}
	if calls != 2 {
		t.Errorf("build called %d times, want 2", calls)
	}
}

func TestAllowedIfacePatterns(t *testing.T) {
	cases := []struct {
		env  string
		want []string
	}{
		{"", defaultAllowedIfaces},
		{"   ", defaultAllowedIfaces},
		{"lo br+ tailscale0", []string{"lo", "br+", "tailscale0"}},
		// Commas separate too: firewall.sh accepts them, so a value that works
		// for one layer has to work for the other.
		{"lo,br+,zt", []string{"lo", "br+", "zt"}},
		// A bare "+" is the iptables match-anything wildcard and would let the
		// WAN through; it is dropped, leaving the strict defaults.
		{"+", defaultAllowedIfaces},
	}

	for _, c := range cases {
		got := allowedIfacePatterns(c.env)
		if len(got) != len(c.want) {
			t.Errorf("allowedIfacePatterns(%q) = %v, want %v", c.env, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("allowedIfacePatterns(%q) = %v, want %v", c.env, got, c.want)
				break
			}
		}
	}
}

func TestIfaceAllowed(t *testing.T) {
	cases := map[string]bool{
		"lo":         true,
		"br0":        true,
		"br1":        true,
		"tun21":      true,
		"tap11":      true,
		"wgs1":       true,
		"eth0":       false,
		"vlan2":      false,
		"ppp0":       false,
		"tailscale0": false,
	}

	for name, want := range cases {
		if got := ifaceAllowed(name, defaultAllowedIfaces); got != want {
			t.Errorf("ifaceAllowed(%q) = %v, want %v", name, got, want)
		}
	}
}

// The guard has to read a pattern exactly as iptables would, or an override
// can open an interface in one layer and leave it shut in the other. A bare
// name is an exact match; only a trailing '+' wildcards.
func TestIfaceAllowedMatchesIptablesGrammar(t *testing.T) {
	cases := []struct {
		patterns []string
		name     string
		want     bool
	}{
		// "eth" is an exact name to iptables, so it must not match eth0 here.
		{[]string{"lo", "br+", "eth"}, "eth0", false},
		{[]string{"lo", "br+", "eth"}, "eth", true},
		{[]string{"eth+"}, "eth0", true},

		// "lo" alone must not pull in a longer name that merely starts with it.
		{[]string{"lo"}, "lo", true},
		{[]string{"lo"}, "lodge0", false},
		{[]string{"lo+"}, "lodge0", true},

		{[]string{"tailscale0"}, "tailscale0", true},
		{[]string{"tailscale0"}, "tailscale1", false},
	}

	for _, c := range cases {
		if got := ifaceAllowed(c.name, c.patterns); got != c.want {
			t.Errorf("ifaceAllowed(%q, %v) = %v, want %v", c.name, c.patterns, got, c.want)
		}
	}
}

// The guarded listener has to swallow refused connections rather than surface
// them as an Accept error, which would take the HTTP server down with it.
func TestGuardedListenerSkipsRefusedConnections(t *testing.T) {
	wan := &net.TCPAddr{IP: net.ParseIP("203.0.113.7"), Port: 8787}
	lan := &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8787}

	inner := &fakeListener{conns: []net.Conn{
		&fakeConn{local: wan},
		&fakeConn{local: wan},
		&fakeConn{local: lan},
	}}
	l := &guardedListener{Listener: inner, guard: testGuard(routerIndex(), nil, defaultAllowedIfaces)}

	c, err := l.Accept()
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if c.LocalAddr().String() != lan.String() {
		t.Errorf("Accept returned %s, want the LAN connection", c.LocalAddr())
	}
	for i, fc := range inner.conns[:2] {
		if !fc.(*fakeConn).closed {
			t.Errorf("refused connection %d was not closed", i)
		}
	}
}

type fakeListener struct {
	conns []net.Conn
	n     int
}

func (l *fakeListener) Accept() (net.Conn, error) {
	if l.n >= len(l.conns) {
		return nil, errors.New("eof")
	}
	c := l.conns[l.n]
	l.n++
	return c, nil
}
func (l *fakeListener) Close() error   { return nil }
func (l *fakeListener) Addr() net.Addr { return &net.TCPAddr{} }

type fakeConn struct {
	net.Conn
	local  net.Addr
	closed bool
}

func (c *fakeConn) LocalAddr() net.Addr  { return c.local }
func (c *fakeConn) RemoteAddr() net.Addr { return &net.TCPAddr{IP: net.ParseIP("198.51.100.1")} }
func (c *fakeConn) Close() error         { c.closed = true; return nil }
