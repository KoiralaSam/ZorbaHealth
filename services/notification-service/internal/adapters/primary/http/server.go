package http

import (
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
// Routes: GET/POST /sms — set this URL on each SMS-enabled DID in VoIP.ms (include ?api_key= matching VOIPMS_API_KEY).
func (s *Server) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sms", s.HandleSMSRequest)
	mux.HandleFunc("POST /sms", s.HandleSMSRequest)

	sharedlogging.Info("notification http server listening", "addr", s.addr)
	if err := http.ListenAndServe(s.addr, otelhttp.NewHandler(mux, "notification-service")); err != nil {
		sharedlogging.Error("notification http server error", err)
	}
}
