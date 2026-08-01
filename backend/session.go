package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	sessionGrace   = 2 * time.Minute
	scrollbackSize = 128 * 1024
	maxSessions    = 16
	writeTimeout   = 15 * time.Second
)

var errNotOwner = errors.New("session belongs to another client")

type session struct {
	id    string
	owner string
	ptmx  *os.File
	proc  *exec.Cmd

	mu     sync.Mutex
	conn   *websocket.Conn
	buf    *ringBuffer
	reaper *time.Timer
	closed bool
}

var (
	sessionsMu sync.Mutex
	sessions   = make(map[string]*session)
)

func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func validSessionID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func acquireSession(id, owner string, cols, rows int) (*session, bool, error) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	if s, ok := sessions[id]; ok {
		if s.owner != owner {
			return nil, false, errNotOwner
		}
		return s, true, nil
	}
	if len(sessions) >= maxSessions {
		return nil, false, fmt.Errorf("too many open sessions (%d)", len(sessions))
	}

	ptmx, proc, err := startShell(cols, rows)
	if err != nil {
		return nil, false, err
	}

	s := &session{id: id, owner: owner, ptmx: ptmx, proc: proc, buf: newRingBuffer(scrollbackSize)}
	sessions[id] = s

	go s.pump()
	go func() {
		proc.Wait()
		s.shutdown("shell exited")
	}()

	return s, false, nil
}

func (s *session) pump() {
	chunk := make([]byte, 32*1024)
	for {
		n, err := s.ptmx.Read(chunk)
		if n > 0 {
			s.broadcast(chunk[:n])
		}
		if err != nil {
			s.shutdown("pty closed")
			return
		}
	}
}

func (s *session) broadcast(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Write(p)

	conn := s.conn
	if conn == nil {
		return
	}
	if err := writeFrame(conn, p); err != nil {
		log.Printf("session %s: write failed, detaching: %v", s.id, err)
		s.conn = nil
		s.armReaperLocked()
		go conn.Close(websocket.StatusInternalError, "write failed")
	}
}

func (s *session) attach(c *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("session already closed")
	}
	if s.reaper != nil {
		s.reaper.Stop()
		s.reaper = nil
	}
	if prev := s.conn; prev != nil && prev != c {
		go prev.Close(websocket.StatusPolicyViolation, "session attached elsewhere")
	}
	s.conn = c

	if replay := s.buf.Snapshot(); len(replay) > 0 {
		if err := writeFrame(c, replay); err != nil {
			s.conn = nil
			s.armReaperLocked()
			return err
		}
	}
	return nil
}

func (s *session) detach(c *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != c {
		return
	}
	s.conn = nil
	if s.closed {
		return
	}
	s.armReaperLocked()
}

func (s *session) armReaperLocked() {
	if s.reaper != nil {
		s.reaper.Stop()
	}
	s.reaper = time.AfterFunc(sessionGrace, func() { s.shutdown("idle timeout") })
}

func (s *session) shutdown(reason string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if s.reaper != nil {
		s.reaper.Stop()
		s.reaper = nil
	}
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()

	sessionsMu.Lock()
	if sessions[s.id] == s {
		delete(sessions, s.id)
	}
	sessionsMu.Unlock()

	if s.proc != nil && s.proc.Process != nil {
		s.proc.Process.Kill()
	}
	s.ptmx.Close()
	if conn != nil {
		conn.Close(websocket.StatusNormalClosure, reason)
	}
	log.Printf("session %s closed (%s)", s.id, reason)
}

func writeFrame(c *websocket.Conn, p []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageBinary, p)
}

type ringBuffer struct {
	data []byte
	pos  int
	full bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{data: make([]byte, size)}
}

func (r *ringBuffer) Write(p []byte) {
	if len(r.data) == 0 || len(p) == 0 {
		return
	}
	if len(p) >= len(r.data) {
		copy(r.data, p[len(p)-len(r.data):])
		r.pos = 0
		r.full = true
		return
	}

	n := copy(r.data[r.pos:], p)
	if n < len(p) {
		copy(r.data, p[n:])
	}
	r.pos += len(p)
	if r.pos >= len(r.data) {
		r.pos -= len(r.data)
		r.full = true
	}
}

func (r *ringBuffer) Snapshot() []byte {
	if !r.full {
		out := make([]byte, r.pos)
		copy(out, r.data[:r.pos])
		return out
	}
	out := make([]byte, 0, len(r.data))
	out = append(out, r.data[r.pos:]...)
	out = append(out, r.data[:r.pos]...)
	return out
}
