package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

var nonceSeq int

func signedRequest(t *testing.T, client string, ts int64) *http.Request {
	t.Helper()
	nonceSeq++
	return signedRequestNonce(t, client, ts, fmt.Sprintf("%032x", nonceSeq))
}

func signedRequestNonce(t *testing.T, client string, ts int64, nonce string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "%s|%d|%s", client, ts, nonce)
	sig := hex.EncodeToString(mac.Sum(nil))

	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.Header.Set("Sec-WebSocket-Protocol",
		fmt.Sprintf("idefix, idefix.auth.%s.%s.%s.%s", client, strconv.FormatInt(ts, 10), nonce, sig))
	return r
}

func TestAuthorised(t *testing.T) {
	secret = []byte("0123456789abcdef")
	spentTokens = &tokenStore{used: make(map[string]time.Time)}
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
		offer := r.Header.Get("Sec-WebSocket-Protocol")
		// signature was issued for client-a
		r.Header.Set("Sec-WebSocket-Protocol", strings.Replace(offer, "client-a", "client-b", 1))
		if _, ok := authorised(r); ok {
			t.Fatal("expected a mismatched client id to be rejected")
		}
	})

	t.Run("missing parameters are rejected", func(t *testing.T) {
		if _, ok := authorised(httptest.NewRequest(http.MethodGet, "/ws", nil)); ok {
			t.Fatal("expected an unsigned request to be rejected")
		}
	})

	t.Run("a credential in the query string is ignored", func(t *testing.T) {
		signed := signedRequest(t, "client-q", now.Unix())
		_, ts, n, sig := credentials(signed)

		url := fmt.Sprintf("/ws?c=%s&t=%s&n=%s&s=%s", "client-q", ts, n, sig)
		if _, ok := authorised(httptest.NewRequest(http.MethodGet, url, nil)); ok {
			t.Fatal("expected the query string to carry no authority")
		}
	})

	t.Run("a token is spent on first use", func(t *testing.T) {
		const nonce = "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"
		if _, ok := authorised(signedRequestNonce(t, "client-replay", now.Unix(), nonce)); !ok {
			t.Fatal("expected the first use of a token to be accepted")
		}
		if _, ok := authorised(signedRequestNonce(t, "client-replay", now.Unix(), nonce)); ok {
			t.Fatal("expected a replayed token to be rejected")
		}
	})

	// Two tabs share one client id and can mint within the same second, so the
	// nonce is what keeps their tokens distinct.
	t.Run("same client and second with distinct nonces both pass", func(t *testing.T) {
		ts := now.Unix()
		if _, ok := authorised(signedRequestNonce(t, "client-tabs", ts, "aa11aa11aa11aa11aa11aa11aa11aa11")); !ok {
			t.Fatal("expected the first tab to be accepted")
		}
		if _, ok := authorised(signedRequestNonce(t, "client-tabs", ts, "bb22bb22bb22bb22bb22bb22bb22bb22")); !ok {
			t.Fatal("expected a second tab minting in the same second to be accepted")
		}
	})

	// hex.DecodeString accepts either case, so a spent token re-cased is still
	// a valid signature and must not read as a fresh one.
	t.Run("a replayed token re-cased is rejected", func(t *testing.T) {
		const nonce = "1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c1c"
		first := signedRequestNonce(t, "client-case", now.Unix(), nonce)
		if _, ok := authorised(first); !ok {
			t.Fatal("expected the first use of a token to be accepted")
		}

		c, ts, n, sig := credentials(signedRequestNonce(t, "client-case", now.Unix(), nonce))
		replay := httptest.NewRequest(http.MethodGet, "/ws", nil)
		replay.Header.Set("Sec-WebSocket-Protocol",
			fmt.Sprintf("idefix, idefix.auth.%s.%s.%s.%s", c, ts, n, strings.ToUpper(sig)))

		if _, ok := authorised(replay); ok {
			t.Fatal("expected an upper-cased replay of a spent token to be rejected")
		}
	})

	t.Run("a credential missing its nonce is rejected", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/ws", nil)
		r.Header.Set("Sec-WebSocket-Protocol", "idefix, idefix.auth.client-a.123.deadbeef")
		if _, ok := authorised(r); ok {
			t.Fatal("expected a three-field credential to be rejected")
		}
	})
}

func TestTokenStoreEvictsExpiredEntries(t *testing.T) {
	s := &tokenStore{used: make(map[string]time.Time)}
	now := time.Now()

	if !s.consume("aaaa", now) {
		t.Fatal("expected an unseen token to be accepted")
	}
	if s.consume("aaaa", now.Add(time.Second)) {
		t.Fatal("expected a token still inside its TTL to be refused")
	}

	s.consume("bbbb", now.Add(2*tokenTTL))
	if len(s.used) != 1 {
		t.Fatalf("store holds %d entries, want the expired one evicted", len(s.used))
	}
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
