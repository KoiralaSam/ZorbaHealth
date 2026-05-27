package memory

import (
	"sync"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
)

var _ outbound.ConnectionRegistry = (*InMemoryConnectionRegistry)(nil)

// InMemoryConnectionRegistry stores active patient WebSocket connections in-process.
type connEntry struct {
	conn outbound.Connection
}

type InMemoryConnectionRegistry struct {
	mu    sync.RWMutex
	conns map[string]connEntry
}

func NewInMemoryConnectionRegistry() outbound.ConnectionRegistry {
	return &InMemoryConnectionRegistry{
		conns: make(map[string]connEntry),
	}
}

func (r *InMemoryConnectionRegistry) Register(patientID string, conn outbound.Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.conns[patientID]; ok {
		existing.conn.Close()
	}
	r.conns[patientID] = connEntry{conn: conn}
}

func (r *InMemoryConnectionRegistry) Unregister(patientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, patientID)
}

func (r *InMemoryConnectionRegistry) ClientIP(patientID string) (string, bool) {
	r.mu.RLock()
	entry, ok := r.conns[patientID]
	r.mu.RUnlock()
	if !ok {
		return "", false
	}
	ip := entry.conn.ClientIP()
	if ip == "" {
		return "", false
	}
	return ip, true
}

func (r *InMemoryConnectionRegistry) Send(patientID string, cmd models.LocationCommand) error {
	r.mu.RLock()
	entry, ok := r.conns[patientID]
	r.mu.RUnlock()
	if !ok {
		return domainerrors.ErrNoActiveConnection
	}
	msg := contracts.WSMessage{
		Type: cmd.Command,
		Data: map[string]any{"session_id": cmd.SessionID},
	}
	return entry.conn.WriteJSON(msg)
}
