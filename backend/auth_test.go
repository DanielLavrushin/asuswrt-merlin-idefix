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
