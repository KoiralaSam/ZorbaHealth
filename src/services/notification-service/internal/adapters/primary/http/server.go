package http

import (
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
)

// Server is the primary HTTP adapter for the notification service (e.g. webhooks).
type Server struct {
	addr          string
	webhookAPIKey string
	svc           inbound.NotificationService
}

// NewServer creates an HTTP server that serves POST /sms (VoIP.ms incoming SMS webhook).
// webhookAPIKey is validated on each request via the api_key query parameter.
func NewServer(addr, webhookAPIKey string, svc inbound.NotificationService) *Server {
	return &Server{
		addr:          addr,
		webhookAPIKey: webhookAPIKey,
		svc:           svc,
	}
}

// Run starts the HTTP server and blocks. Typically run in a goroutine.
// Route: POST /sms — configure your public URL (e.g. https://xxx.ngrok-free.dev/sms) in VoIP.ms so they can reach this webhook.
func (s *Server) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sms", s.HandleSMSRequest)

	log.Printf("HTTP server listening on %s", s.addr)
	if err := http.ListenAndServe(s.addr, mux); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
