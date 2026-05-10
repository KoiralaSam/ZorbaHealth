package outbound

import (
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
)

// ConnectionRegistry is the secondary port for WebSocket connection management.
// The core calls this to push commands to connected apps.
// Implementation lives in the HTTP adapter — the core never touches net/http.
type ConnectionRegistry interface {
	Register(patientID string, conn Connection)
	Unregister(patientID string)
	Send(patientID string, cmd models.LocationCommand) error
}

// Connection abstracts the underlying WebSocket connection.
// Lets us swap gorilla/websocket for another implementation without changing core.
type Connection interface {
	WriteJSON(v any) error
	Close() error
}
