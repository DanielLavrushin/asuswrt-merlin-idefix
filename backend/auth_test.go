package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func signedRequest(t *testing.T, client string, ts int64) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "%s|%d", client, ts)
	sig := hex.EncodeToString(mac.Sum(nil))

	url := fmt.Sprintf("/ws?c=%s&t=%s&s=%s", client, strconv.FormatInt(ts, 10), sig)
	return httptest.NewRequest(http.MethodGet, url, nil)
}

func TestAuthorised(t *testing.T) {
	secret = []byte("0123456789abcdef")
	now := time.Now()

	t.Run("fresh token is accepted", func(t *testing.T) {
		owner, ok := authorised(signedRequest(t, "client-a", now.Unix()))
		if !ok {
			t.Fatal("expected a fresh token to be accepted")
		}
		if owner != "client-a" {
			t.Fatalf("owner = %q, want %q", owner, "client-a")
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		if _, ok := authorised(signedRequest(t, "client-a", now.Add(-3*time.Minute).Unix())); ok {
			t.Fatal("expected an expired token to be rejected")
		}
	})

	t.Run("future token is rejected", func(t *testing.T) {
		// Without an upper bound this one never ages out.
		if _, ok := authorised(signedRequest(t, "client-a", now.Add(24*time.Hour).Unix())); ok {
			t.Fatal("expected a future-dated token to be rejected")
		}
	})

	t.Run("small clock skew is tolerated", func(t *testing.T) {
		if _, ok := authorised(signedRequest(t, "client-a", now.Add(10*time.Second).Unix())); !ok {
			t.Fatal("expected a slightly-ahead token to be accepted")
		}
	})

	t.Run("forged signature is rejected", func(t *testing.T) {
		r := signedRequest(t, "client-a", now.Unix())
		q := r.URL.Query()
		q.Set("c", "client-b") // signature was issued for client-a
		r.URL.RawQuery = q.Encode()
		if _, ok := authorised(r); ok {
			t.Fatal("expected a mismatched client id to be rejected")
		}
	})

	t.Run("missing parameters are rejected", func(t *testing.T) {
		if _, ok := authorised(httptest.NewRequest(http.MethodGet, "/ws", nil)); ok {
			t.Fatal("expected an unsigned request to be rejected")
		}
	})
}

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{"no origin at all", "192.168.1.1:8787", "", true},
		{"router ui on the default port", "192.168.1.1:8787", "http://192.168.1.1", true},
		{"router ui on the tls port", "192.168.1.1:8787", "https://192.168.1.1:8443", true},
		{"hostname instead of address", "router.asus.com:8787", "http://router.asus.com", true},
		{"asus hostname over tls", "www.asusrouter.com:8787", "https://www.asusrouter.com:8443", true},
		{"ddns name", "myrouter.asuscomm.com:8787", "https://myrouter.asuscomm.com:8443", true},
		{"hostname case is ignored", "Router.Asus.Com:8787", "http://router.asus.com", true},
		{"ipv6 literal", "[fd00::1]:8787", "http://[fd00::1]:8443", true},
		{"host without a port", "192.168.1.1", "http://192.168.1.1", true},
		{"attacker page", "192.168.1.1:8787", "https://evil.example", false},
		{"lookalike suffix", "192.168.1.1:8787", "http://192.168.1.1.evil.example", false},
		{"opaque origin", "192.168.1.1:8787", "null", false},
		{"origin without a host", "192.168.1.1:8787", "file://", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/ws", nil)
			r.Host = tc.host
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := originAllowed(r); got != tc.want {
				t.Fatalf("originAllowed(host=%q, origin=%q) = %v, want %v", tc.host, tc.origin, got, tc.want)
			}
		})
	}
}
