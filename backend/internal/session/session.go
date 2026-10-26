// Package session manages per-connection state: the MUD connection,
// scrollback history and lifecycle.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/tollofsen/bazooka/backend/internal/mud"
	"github.com/tollofsen/bazooka/backend/internal/router"
	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

const maxScrollbackChunks = 2000

var (
	ErrNotFound = errors.New("session not found")
	ErrExists   = errors.New("session already exists")
	ErrClosed   = errors.New("session is not connected")
)

// Session is one MUD connection plus its state.
type Session struct {
	ID      string
	Host    string
	Port    int
	Started time.Time

	conn   *mud.Conn
	router *router.Router

	mu         sync.Mutex
	connected  bool
	scrollback []*pb.Output
}

// Manager owns all sessions.
type Manager struct {
	router *router.Router

	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager creates a Manager publishing output on r.
func NewManager(r *router.Router) *Manager {
	return &Manager{router: r, sessions: make(map[string]*Session)}
}

// Connect dials the MUD and starts reading from it. If id is empty one is generated.
func (m *Manager) Connect(ctx context.Context, id, host string, port int) (*Session, error) {
	if host == "" || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid host/port %q:%d", host, port)
	}
	if id == "" {
		id = newID()
	}
	m.mu.Lock()
	if _, ok := m.sessions[id]; ok {
		m.mu.Unlock()
		return nil, ErrExists
	}
	// Reserve the ID while dialing so concurrent calls cannot race.
	s := &Session{ID: id, Host: host, Port: port, Started: time.Now(), router: m.router}
	m.sessions[id] = s
	m.mu.Unlock()

	conn, err := mud.Dial(ctx, host, port)
	if err != nil {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		return nil, err
	}
	s.mu.Lock()
	s.conn = conn
	s.connected = true
	s.mu.Unlock()
	s.emit(pb.OutputKind_OUTPUT_KIND_SYSTEM, fmt.Sprintf("Connected to %s:%d\n", host, port))
	go s.run()
	return s, nil
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

// List returns info on all sessions.
func (m *Manager) List() []*pb.SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*pb.SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.Info())
	}
	return out
}

// Disconnect closes and removes a session.
func (m *Manager) Disconnect(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	s.close()
	return nil
}

// CloseAll disconnects every session.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	all := m.sessions
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, s := range all {
		s.close()
	}
}

// Info returns the session's protobuf description.
func (s *Session) Info() *pb.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &pb.SessionInfo{
		SessionId:     s.ID,
		Host:          s.Host,
		Port:          uint32(s.Port),
		Connected:     s.connected,
		StartedUnixMs: s.Started.UnixMilli(),
	}
}

// Send writes a command to the MUD and echoes it to all subscribers.
func (s *Session) Send(command, source string) error {
	s.mu.Lock()
	conn, ok := s.conn, s.connected
	s.mu.Unlock()
	if !ok || conn == nil {
		return ErrClosed
	}
	if err := conn.Send(command); err != nil {
		return err
	}
	prefix := ""
	if source != "" {
		prefix = "[" + source + "] "
	}
	s.emit(pb.OutputKind_OUTPUT_KIND_COMMAND, prefix+command+"\n")
	return nil
}

// Scrollback returns the last n lines of history as output chunks.
func (s *Session) Scrollback(n int) []*pb.Output {
	if n <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Walk backwards counting newlines until n lines are covered.
	lines, start := 0, len(s.scrollback)
	for start > 0 && lines < n {
		start--
		lines += strings.Count(s.scrollback[start].Text, "\n")
	}
	return append([]*pb.Output(nil), s.scrollback[start:]...)
}

func (s *Session) run() {
	err := s.conn.ReadLoop(func(text string) {
		s.emit(pb.OutputKind_OUTPUT_KIND_GAME, text)
	})
	msg := "Connection closed by server\n"
	if err != nil && !errors.Is(err, io.EOF) {
		msg = fmt.Sprintf("Connection lost: %v\n", err)
	}
	s.mu.Lock()
	wasConnected := s.connected
	s.connected = false
	s.mu.Unlock()
	if wasConnected {
		s.emit(pb.OutputKind_OUTPUT_KIND_SYSTEM, msg)
	}
}

func (s *Session) close() {
	s.mu.Lock()
	was := s.connected
	s.connected = false
	conn := s.conn
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	if was {
		s.emit(pb.OutputKind_OUTPUT_KIND_SYSTEM, "Disconnected\n")
	}
}

func (s *Session) emit(kind pb.OutputKind, text string) {
	o := &pb.Output{SessionId: s.ID, Kind: kind, Text: text, TimestampUnixMs: time.Now().UnixMilli()}
	s.mu.Lock()
	s.scrollback = append(s.scrollback, o)
	if len(s.scrollback) > maxScrollbackChunks {
		s.scrollback = s.scrollback[len(s.scrollback)-maxScrollbackChunks:]
	}
	s.mu.Unlock()
	s.router.Publish(o)
}

func newID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
