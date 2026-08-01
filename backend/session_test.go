package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestRingBufferShorterThanCapacity(t *testing.T) {
	r := newRingBuffer(16)
	r.Write([]byte("abc"))
	r.Write([]byte("def"))
	if got := string(r.Snapshot()); got != "abcdef" {
		t.Fatalf("got %q, want %q", got, "abcdef")
	}
}

func TestRingBufferExactFill(t *testing.T) {
	r := newRingBuffer(6)
	r.Write([]byte("abc"))
	r.Write([]byte("def"))
	if got := string(r.Snapshot()); got != "abcdef" {
		t.Fatalf("got %q, want %q", got, "abcdef")
	}
}

func TestRingBufferWrap(t *testing.T) {
	r := newRingBuffer(6)
	r.Write([]byte("abcd"))
	r.Write([]byte("efgh"))
	if got := string(r.Snapshot()); got != "cdefgh" {
		t.Fatalf("got %q, want %q", got, "cdefgh")
	}
}

func TestRingBufferOversizedWrite(t *testing.T) {
	r := newRingBuffer(4)
	r.Write([]byte("0123456789"))
	if got := string(r.Snapshot()); got != "6789" {
		t.Fatalf("got %q, want %q", got, "6789")
	}
}

func TestValidSessionID(t *testing.T) {
	for _, id := range []string{"0123456789abcdef", "a-b_c-d_e", strings.Repeat("a", 64)} {
		if !validSessionID(id) {
			t.Errorf("expected %q to be valid", id)
		}
	}
	for _, id := range []string{"", "short", strings.Repeat("a", 65), "has spaces", "semi;colon", "../../etc"} {
		if validSessionID(id) {
			t.Errorf("expected %q to be rejected", id)
		}
	}
}

// sessionServer serves the post-auth half of the real handler.
func sessionServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols:       []string{"idefix"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		c.SetReadLimit(maxMessageBytes)
		serveSession(r.Context(), c, r.URL.Query().Get("sid"), r.URL.Query().Get("owner"), r.RemoteAddr)
	}))
}

const testOwner = "test-client-token"

func dial(t *testing.T, srv *httptest.Server, sid string) *websocket.Conn {
	t.Helper()
	c, err := dialAs(srv, sid, testOwner)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return c
}

func dialAs(srv *httptest.Server, sid, owner string) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?sid=" + sid + "&owner=" + owner
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{Subprotocols: []string{"idefix"}})
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(maxMessageBytes)
	return c, nil
}

// readUntil accumulates pty output until it contains want, or the deadline hits.
func readUntil(t *testing.T, c *websocket.Conn, want string, timeout time.Duration) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var sb strings.Builder
	for {
		_, rdr, err := c.Reader(ctx)
		if err != nil {
			t.Fatalf("waiting for %q, read failed after %q: %v", want, sb.String(), err)
		}
		data, err := io.ReadAll(rdr)
		if err != nil {
			t.Fatalf("read payload: %v", err)
		}
		sb.Write(data)
		if strings.Contains(sb.String(), want) {
			return sb.String()
		}
	}
}

func send(t *testing.T, c *websocket.Conn, s string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(s)); err != nil {
		t.Fatalf("write %q: %v", s, err)
	}
}

// TestSessionSurvivesDisconnect is the Safari case: the browser's socket goes
// away mid-session and comes back with the same session id. The shell must be
// the same one, and its recent output must be replayed.
func TestSessionSurvivesDisconnect(t *testing.T) {
	srv := sessionServer(t)
	defer srv.Close()

	const sid = "test-session-survives"

	c1 := dial(t, srv, sid)
	send(t, c1, "MARK=idefix-lives\n")
	send(t, c1, "echo first-run-$MARK\n")
	readUntil(t, c1, "first-run-idefix-lives", 10*time.Second)

	sessionsMu.Lock()
	sess := sessions[sid]
	sessionsMu.Unlock()
	if sess == nil {
		t.Fatal("session was not registered")
	}
	pid := sess.proc.Process.Pid

	// Same shape as a Safari back/forward-cache suspension.
	c1.Close(websocket.StatusGoingAway, "WebSocket is closed due to suspension.")

	// Output produced while nobody is attached must not be lost.
	time.Sleep(200 * time.Millisecond)
	sess.ptmx.Write([]byte("echo while-detached\n"))
	time.Sleep(300 * time.Millisecond)

	c2 := dial(t, srv, sid)
	defer c2.Close(websocket.StatusNormalClosure, "")

	replay := readUntil(t, c2, "while-detached", 10*time.Second)
	if !strings.Contains(replay, "first-run-idefix-lives") {
		t.Errorf("scrollback from before the drop was not replayed; got %q", replay)
	}

	sessionsMu.Lock()
	resumed := sessions[sid]
	sessionsMu.Unlock()
	if resumed != sess {
		t.Fatal("reconnect created a new session instead of resuming")
	}
	if resumed.proc.Process.Pid != pid {
		t.Fatalf("shell was restarted: pid %d -> %d", pid, resumed.proc.Process.Pid)
	}

	// The shell kept its state across the drop.
	send(t, c2, "echo resumed-$MARK\n")
	readUntil(t, c2, "resumed-idefix-lives", 10*time.Second)
}

// TestByeClosesSession covers the deliberate close: no lingering shell.
func TestByeClosesSession(t *testing.T) {
	srv := sessionServer(t)
	defer srv.Close()

	const sid = "test-session-bye-xx"

	c := dial(t, srv, sid)
	send(t, c, "echo ready\n")
	readUntil(t, c, "ready", 10*time.Second)

	send(t, c, `{"type":"bye"}`)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		sessionsMu.Lock()
		_, still := sessions[sid]
		sessionsMu.Unlock()
		if !still {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("session still registered after bye")
}

// TestLargePasteDoesNotKillSession guards the read limit: a paste bigger than
// the library's 32 KiB default used to end the connection.
func TestLargePasteDoesNotKillSession(t *testing.T) {
	srv := sessionServer(t)
	defer srv.Close()

	const sid = "test-session-bigpaste"

	c := dial(t, srv, sid)
	defer c.Close(websocket.StatusNormalClosure, "")

	send(t, c, "cat > /dev/null <<'IDEFIX_EOF'\n")
	send(t, c, strings.Repeat("x", 200*1024)+"\n")
	send(t, c, "IDEFIX_EOF\n")
	send(t, c, "echo survived-the-paste\n")

	readUntil(t, c, "survived-the-paste", 15*time.Second)
}

// TestSessionIDIsNotACapability: knowing another client's session id is not
// enough to attach to its shell.
func TestSessionIDIsNotACapability(t *testing.T) {
	srv := sessionServer(t)
	defer srv.Close()

	const sid = "victim-session-01"

	victim := dial(t, srv, sid)
	defer victim.Close(websocket.StatusNormalClosure, "")
	send(t, victim, "echo victim-ready\n")
	readUntil(t, victim, "victim-ready", 10*time.Second)

	attacker, err := dialAs(srv, sid, "some-other-client")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer attacker.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := attacker.Reader(ctx); err == nil {
		t.Fatal("expected the attach to be refused")
	} else if !strings.Contains(err.Error(), "belongs to another client") {
		t.Fatalf("refused for the wrong reason: %v", err)
	}

	// The victim's shell is untouched.
	send(t, victim, "echo victim-still-here\n")
	readUntil(t, victim, "victim-still-here", 10*time.Second)
}

// TestNewSessionIDPerClient: two clients without a usable id get their own shells.
func TestDistinctSessionsPerID(t *testing.T) {
	srv := sessionServer(t)
	defer srv.Close()

	a := dial(t, srv, "session-alpha-01")
	defer a.Close(websocket.StatusNormalClosure, "")
	b := dial(t, srv, "session-beta-001")
	defer b.Close(websocket.StatusNormalClosure, "")

	send(t, a, "echo alpha-here\n")
	readUntil(t, a, "alpha-here", 10*time.Second)

	sessionsMu.Lock()
	sa, sb := sessions["session-alpha-01"], sessions["session-beta-001"]
	sessionsMu.Unlock()

	if sa == nil || sb == nil {
		t.Fatal("expected both sessions to be registered")
	}
	if sa.proc.Process.Pid == sb.proc.Process.Pid {
		t.Fatal("distinct session ids shared a shell")
	}
}
