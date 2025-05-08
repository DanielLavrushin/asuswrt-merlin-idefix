// idefix.go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

var port int

func main() {
	flag.IntVar(&port, "port", 8787, "listen port")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	mux.Handle("/", http.FileServer(http.Dir("./public"))) // optional UI

	addr := ":" + strconv.Itoa(port)
	fmt.Printf("⚡ WebSocket PTY listening on %s/ws\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	if !authorised(r) {
		fmt.Println("Unauthorized access")
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		w.WriteHeader(401)
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
	tok, _ := r.Cookie("asus_token")
	fmt.Println("Token:", tok)

	if tok == nil {
		return false
	}
	if tok.Value == "" {
		return false
	}
	if tokenValid(tok.Value) {
		return true
	}
	return false
}

var routerIP = "192.168.1.1"

func tokenValid(token string) bool {
	u := url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", "80"),
		Path: "/ajax_status.xml"}

	if httpsOnly() {
		u.Scheme = "https"
		u.Host = net.JoinHostPort("127.0.0.1", "443")
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false
	}

	req.Header.Set("Cookie", "asus_token="+token)
	req.Host = routerIP

	cli := http.DefaultClient
	if u.Scheme == "https" {
		cli = &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func httpsOnly() bool {
	// quick probe – open port 80; if closed, assume HTTPS-only
	c, err := net.DialTimeout("tcp", "127.0.0.1:80", time.Second)
	if err != nil {
		return true
	}
	c.Close()
	return false
}

type wsWriter struct{ *websocket.Conn }

func (w wsWriter) Write(p []byte) (int, error) {
	return len(p), w.Conn.Write(context.Background(), websocket.MessageBinary, p)
}
