// idefix.go
package main

import (
	"bufio"
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

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

const (
	plainAddr = ":8787"
	tlsAddr   = ":8787"
	certFile  = "/jffs/addons/idefix/cert.pem"
	keyFile   = "/jffs/addons/idefix/key.pem"
)

var (
	port   int
	secret []byte
)

const tlsHandshakeByte byte = 0x16

func ServeOnSamePort(addr, cert, key string, handler http.Handler) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	plain := &http.Server{Handler: handler}

	tlsCfg, _ := tls.LoadX509KeyPair(cert, key)
	tlsSrv := &http.Server{
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCfg},
		},
	}

	go tlsSrv.Serve(&tlsListener{ln})

	return plain.Serve(&plainListener{ln})
}

type peekConn struct {
	net.Conn
	br *bufio.Reader
}

func (c *peekConn) Read(b []byte) (int, error) { return c.br.Read(b) }

type plainListener struct{ net.Listener }
type tlsListener struct{ net.Listener }

func (l *plainListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		pc := &peekConn{Conn: c, br: bufio.NewReader(c)}
		if b, _ := pc.br.Peek(1); len(b) == 1 && b[0] != tlsHandshakeByte {
			return pc, nil // plain
		}
		c.Close()
	}
}

func (l *tlsListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		pc := &peekConn{Conn: c, br: bufio.NewReader(c)}
		if b, _ := pc.br.Peek(1); len(b) == 1 && b[0] == tlsHandshakeByte {
			return pc, nil // TLS
		}
		c.Close()
	}
}

func main() {
	flag.IntVar(&port, "port", 8787, "listen port")
	flag.Parse()

	var sec_path = "/jffs/addons/idefix/sec.key"

	raw, err := os.ReadFile(sec_path)
	if err != nil {
		panic("missing " + sec_path + " file")
	}
	secret, _ = hex.DecodeString(string(bytes.TrimSpace(raw)))

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)

	log.Printf("▶︎  WS + WSS on :8787")
	if err := ServeOnSamePort(":8787", certFile, keyFile, mux); err != nil {
		log.Fatal(err)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	if !authorised(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       []string{"idefix"},
		CompressionMode:    websocket.CompressionContextTakeover,
		InsecureSkipVerify: true,
	})
	if err != nil {
		fmt.Println("WS upgrade failed:", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	ptmx, proc, err := startShell(80, 24)
	fmt.Printf("Connected to shell from %s\n", r.RemoteAddr)
	if err != nil {
		c.Close(websocket.StatusInternalError, err.Error())
		return
	}
	defer func() {
		proc.Process.Kill()
		ptmx.Close()
	}()

	go func() {
		_, _ = io.Copy(wsWriter{c}, ptmx)
	}()

	ctx := r.Context()
	for {
		msgType, rdr, err := c.Reader(ctx)
		if err != nil {
			break
		}
		data, _ := io.ReadAll(rdr)

		if msgType == websocket.MessageText && len(data) > 0 && data[0] == '{' {
			var ctrl struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
				resizePTY(ptmx, ctrl.Cols, ctrl.Rows)
				continue
			}
		}

		_, _ = ptmx.Write(data)
	}
}

func startShell(cols, rows int) (ptmx *os.File, cmd *exec.Cmd, err error) {
	cmd = exec.Command("/bin/sh") // BusyBox ash on Merlin
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

type wsWriter struct{ *websocket.Conn }

func (w wsWriter) Write(p []byte) (int, error) {
	return len(p), w.Conn.Write(context.Background(), websocket.MessageBinary, p)
}
