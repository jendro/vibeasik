package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// SSESession holds connection state for an active MCP SSE client
type SSESession struct {
	ID       string
	Messages chan []byte
	Done     chan struct{}
}

// SessionManager coordinates active SSE stream connections
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SSESession
}

var globalSessions = &SessionManager{
	sessions: make(map[string]*SSESession),
}

// GetSessionManager returns singleton SSE session manager
func GetSessionManager() *SessionManager {
	return globalSessions
}

// CreateSession allocates a new SSE session with a random unique ID
func (sm *SessionManager) CreateSession() *SSESession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)

	session := &SSESession{
		ID:       id,
		Messages: make(chan []byte, 64),
		Done:     make(chan struct{}),
	}
	sm.sessions[id] = session
	return session
}

// GetSession returns an active session by ID
func (sm *SessionManager) GetSession(id string) (*SSESession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// RemoveSession cleans up an SSE session
func (sm *SessionManager) RemoveSession(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sessions[id]; ok {
		select {
		case <-s.Done:
		default:
			close(s.Done)
		}
		delete(sm.sessions, id)
	}
}
