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

	"github.com/coder/websocket"
	"github.com/creack/pty"
	"github.com/soheilhy/cmux"
)

const (
	certFile = "/etc/cert.pem"
	keyFile  = "/etc/key.pem"
)

var (
	port   int
	secret []byte
)

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

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	m := cmux.New(ln)

	tlsCfg, _ := tls.LoadX509KeyPair(certFile, keyFile)
	tlsL := tls.NewListener(m.Match(cmux.TLS()), &tls.Config{
		Certificates: []tls.Certificate{tlsCfg},
	})

	httpL := m.Match(cmux.HTTP1Fast())

	go (&http.Server{Handler: mux}).Serve(httpL)
	go (&http.Server{Handler: mux}).Serve(tlsL)

	log.Printf("🐾 Idefix Terminal Server on :%d", port)
	log.Fatal(m.Serve())
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	if !authorised(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		fmt.Println("Forbidden")
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

	go func() {
		proc.Wait()
		c.Close(websocket.StatusNormalClosure, "shell exited")
	}()

	ctx := r.Context()
	for {
		msgType, rdr, err := c.Reader(ctx)
		if err != nil {
			fmt.Println("WS read error:", err)
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
