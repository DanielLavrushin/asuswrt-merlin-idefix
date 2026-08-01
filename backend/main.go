// idefix.go
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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

// maxMessageBytes caps a single inbound frame; generous enough that pasting a
// script into the terminal doesn't tear the connection down.
const maxMessageBytes = 1 << 20

var (
	port   int
	secret []byte
)

func main() {

	flag.IntVar(&port, "port", 8787, "listen port")
	flag.Parse()

	initTZ()
	setupLogging()

	const sec_path = "/jffs/addons/idefix/sec.key"

	raw, err := os.ReadFile(sec_path)
	if err != nil {
		panic("missing " + sec_path + " file")
	}
	secret, _ = hex.DecodeString(string(bytes.TrimSpace(raw)))

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	m := cmux.New(ln)

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

func wsHandler(w http.ResponseWriter, r *http.Request) {

	if !authorised(r) {
		log.Printf("unauthorised from %s – c=%q  t=%q", r.RemoteAddr, r.URL.Query().Get("c"), r.URL.Query().Get("t"))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	log.Printf("authorised client %s (c=%s)", r.RemoteAddr, r.URL.Query().Get("c"))

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{"idefix"},
		CompressionMode:    websocket.CompressionContextTakeover,
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("WS upgrade failed from %s: %v", r.RemoteAddr, err)
		return
	}

	// Default is 32 KiB, which turns a large paste into a fatal read error.
	c.SetReadLimit(maxMessageBytes)

	log.Printf("WS connected (%s)", r.RemoteAddr)

	serveSession(r.Context(), c, r.URL.Query().Get("sid"), r.RemoteAddr)
}

// serveSession attaches c to its shell — resuming an existing one when the
// client brings back a known session id — and pumps browser input into the pty
// until the socket drops.
func serveSession(ctx context.Context, c *websocket.Conn, sid, remote string) {
	if !validSessionID(sid) {
		if sid != "" {
			log.Printf("ignoring malformed session id from %s", remote)
		}
		sid = newSessionID()
	}

	sess, resumed, err := acquireSession(sid, 80, 24)
	if err != nil {
		log.Printf("session error for %s: %v", remote, err)
		c.Close(websocket.StatusInternalError, err.Error())
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
			// A Safari back/forward-cache suspension shows up here as a normal
			// going-away close; the shell stays alive for sessionGrace so the
			// client can pick it up again.
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
					// Tab closed on purpose — don't leave the shell waiting out
					// the grace period.
					sess.shutdown("client closed session")
					return
				}
			}
		}

		_, _ = sess.ptmx.Write(data)
	}
}

func startShell(cols, rows int) (ptmx *os.File, cmd *exec.Cmd, err error) {
	cmd = exec.Command("/bin/sh") // BusyBox ash on Merlin
	log.Printf("Starting shell: %s", cmd.String())
	winsz := &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}
	ptmx, err = pty.StartWithSize(cmd, winsz)
	return
}

func resizePTY(f *os.File, cols, rows int) {
	pty.Setsize(f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func authorised(r *http.Request) bool {
	q := r.URL.Query()
	c := q.Get("c")
	s := q.Get("s")
	t := q.Get("t")

	if c == "" || s == "" || t == "" {
		return false
	}

	ts, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 2*time.Minute {
		fmt.Println("token expired")
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(c))
	mac.Write([]byte("|"))
	mac.Write([]byte(t))
	expected := mac.Sum(nil)

	sent, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	if !hmac.Equal(expected, sent) {
		fmt.Println("bad signature")
		return false
	}
	return true
}
