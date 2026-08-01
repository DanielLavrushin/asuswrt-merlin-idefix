package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/coder/websocket"
	"github.com/creack/pty"
	"github.com/soheilhy/cmux"
)

var certPaths = []struct{ cert, key string }{
	{"/etc/cert.pem", "/etc/key.pem"},
	{"/tmp/etc/cert.pem", "/tmp/etc/key.pem"},
	{"/jffs/.cert/cert.pem", "/jffs/.cert/key.pem"},
}

func loadCert() (tls.Certificate, error) {
	for _, p := range certPaths {
		cert, err := tls.LoadX509KeyPair(p.cert, p.key)
		if err == nil {
			log.Printf("Using TLS: %s", p.cert)
			return cert, nil
		}
	}
	return tls.Certificate{}, fmt.Errorf("no certs found")
}

const maxMessageBytes = 1 << 20

const (
	tokenTTL       = 2 * time.Minute
	maxClockSkew   = 30 * time.Second
	minSecretBytes = 32
)

var (
	port   int
	secret []byte
)

func main() {

	flag.IntVar(&port, "port", 8787, "listen port")
	flag.Parse()

	// Logging first, so that a timezone we cannot resolve is reported to the
	// system log rather than to a stderr nobody reads. Every message is
	// stamped when it is written, so the ones after this still come out right.
	setupLogging()
	initTZ()

	const sec_path = "/jffs/addons/idefix/sec.key"

	raw, err := os.ReadFile(sec_path)
	if err != nil {
		panic("missing " + sec_path + " file")
	}
	secret, err = hex.DecodeString(string(bytes.TrimSpace(raw)))
	if err != nil {
		log.Fatalf("%s is not valid hex: %v", sec_path, err)
	}
	if len(secret) < minSecretBytes {
		log.Fatalf("%s holds %d bytes of key material, need at least %d", sec_path, len(secret), minSecretBytes)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// Second line of defence behind scripts/firewall.sh: a connection that
	// arrived on a WAN address is closed before cmux or TLS ever sees it.
	guard := newLANGuard()
	log.Printf("serving only on interfaces: %v", guard.patterns)
	if idx, err := guard.index(true); err != nil {
		log.Printf("cannot enumerate interfaces (%v); allowing private addresses only", err)
	} else {
		log.Printf("local addresses: %s", idx.inventory(guard.patterns))
	}

	m := cmux.New(&guardedListener{Listener: ln, guard: guard})

	tlsCfg, err := loadCert()
	if err != nil {
		log.Fatal(err)
	}

	tlsL := tls.NewListener(m.Match(cmux.TLS()), &tls.Config{
		Certificates: []tls.Certificate{tlsCfg},
		NextProtos:   []string{"h2", "http/1.1"},
	})

	httpL := m.Match(cmux.Any())

	go (&http.Server{Handler: mux}).Serve(httpL)
	go (&http.Server{Handler: mux}).Serve(tlsL)

	log.Printf("🐾 Idefix Terminal Server on :%d", port)
	log.Fatal(m.Serve())
}

func originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	page := u.Hostname()
	if page == "" {
		return false
	}
	return strings.EqualFold(page, (&url.URL{Host: r.Host}).Hostname())
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	if !originAllowed(r) {
		log.Printf("rejected cross-origin handshake from %s – origin=%q host=%q", r.RemoteAddr, r.Header.Get("Origin"), r.Host)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	owner, ok := authorised(r)
	if !ok {
		log.Printf("unauthorised from %s", r.RemoteAddr)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	log.Printf("authorised client %s (c=%s)", r.RemoteAddr, owner)

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{"idefix"},
		CompressionMode:    websocket.CompressionContextTakeover,
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WS upgrade failed from %s: %v", r.RemoteAddr, err)
		return
	}

	c.SetReadLimit(maxMessageBytes)

	log.Printf("WS connected (%s)", r.RemoteAddr)

	serveSession(r.Context(), c, r.URL.Query().Get("sid"), owner, r.RemoteAddr)
}

func serveSession(ctx context.Context, c *websocket.Conn, sid, owner, remote string) {
	if !validSessionID(sid) {
		if sid != "" {
			log.Printf("ignoring malformed session id from %s", remote)
		}
		sid = newSessionID()
	}

	sess, resumed, err := acquireSession(sid, owner, 80, 24)
	if err != nil {
		log.Printf("session error for %s: %v", remote, err)
		if errors.Is(err, errNotOwner) {
			c.Close(websocket.StatusPolicyViolation, "session belongs to another client")
		} else {
			c.Close(websocket.StatusInternalError, err.Error())
		}
		return
	}

	if resumed {
		log.Printf("re-attached to session %s (pid=%d) from %s", sid, sess.proc.Process.Pid, remote)
	} else {
		log.Printf("new session %s (pid=%d) for %s", sid, sess.proc.Process.Pid, remote)
	}

	if err := sess.attach(c); err != nil {
		log.Printf("attach to session %s failed: %v", sid, err)
		c.Close(websocket.StatusInternalError, "attach failed")
		return
	}

	defer func() {
		sess.detach(c)
		c.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		msgType, rdr, err := c.Reader(ctx)
		if err != nil {
			log.Printf("WS read error (%s, session %s): %v", remote, sid, err)
			return
		}
		data, err := io.ReadAll(rdr)
		if err != nil {
			log.Printf("WS payload error (%s, session %s): %v", remote, sid, err)
			return
		}

		if msgType == websocket.MessageText && len(data) > 0 && data[0] == '{' {
			var ctrl struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(data, &ctrl) == nil {
				switch ctrl.Type {
				case "resize":
					if ctrl.Cols > 0 && ctrl.Rows > 0 {
						resizePTY(sess.ptmx, ctrl.Cols, ctrl.Rows)
					}
					continue
				case "bye":
					sess.shutdown("client closed session")
					return
				}
			}
		}

		_, _ = sess.ptmx.Write(data)
	}
}

func startShell(cols, rows int) (ptmx *os.File, cmd *exec.Cmd, err error) {
	cmd = exec.Command("/bin/sh")
	log.Printf("Starting shell: %s", cmd.String())
	winsz := &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}
	ptmx, err = pty.StartWithSize(cmd, winsz)
	return
}

func resizePTY(f *os.File, cols, rows int) {
	pty.Setsize(f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

const authProtoPrefix = "idefix.auth."

func credentials(r *http.Request) (client, ts, nonce, sig string) {
	for _, header := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, offer := range strings.Split(header, ",") {
			offer = strings.TrimSpace(offer)
			if !strings.HasPrefix(offer, authProtoPrefix) {
				continue
			}
			parts := strings.Split(strings.TrimPrefix(offer, authProtoPrefix), ".")
			if len(parts) != 4 {
				return "", "", "", ""
			}
			return parts[0], parts[1], parts[2], parts[3]
		}
	}
	return "", "", "", ""
}

type tokenStore struct {
	mu   sync.Mutex
	used map[string]time.Time
}

var spentTokens = &tokenStore{used: make(map[string]time.Time)}

func (s *tokenStore) consume(sig string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for seen, at := range s.used {
		if now.Sub(at) > tokenTTL+maxClockSkew {
			delete(s.used, seen)
		}
	}

	if _, spent := s.used[sig]; spent {
		return false
	}
	s.used[sig] = now
	return true
}

func authorised(r *http.Request) (string, bool) {
	c, t, n, s := credentials(r)

	if c == "" || s == "" || t == "" || n == "" {
		return "", false
	}

	ts, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return "", false
	}

	age := time.Since(time.Unix(ts, 0))
	if age > tokenTTL {
		log.Printf("token expired (%s old)", age.Truncate(time.Second))
		return "", false
	}
	if age < -maxClockSkew {
		log.Printf("token rejected: timestamp %s in the future", (-age).Truncate(time.Second))
		return "", false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(c))
	mac.Write([]byte("|"))
	mac.Write([]byte(t))
	mac.Write([]byte("|"))
	mac.Write([]byte(n))
	expected := mac.Sum(nil)

	sent, err := hex.DecodeString(s)
	if err != nil {
		return "", false
	}
	if !hmac.Equal(expected, sent) {
		log.Printf("bad signature from %s", r.RemoteAddr)
		return "", false
	}
	if !spentTokens.consume(s, time.Now()) {
		log.Printf("token replayed from %s", r.RemoteAddr)
		return "", false
	}
	return c, true
}
