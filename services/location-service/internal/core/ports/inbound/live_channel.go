package inbound

// PatientLiveChannel is a push channel to the patient's client (e.g. WebSocket).
// Primary adapters depend on this type only, not on outbound ports.
type PatientLiveChannel interface {
	WriteJSON(v any) error
	Close() error
}
