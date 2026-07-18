package http

import (
	"net/http"

	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
)

// HandleSMSRequest handles GET/POST /sms (VoIP.ms incoming SMS URL callback).
// Validates the api_key query parameter against VOIPMS_API_KEY.
func (s *Server) HandleSMSRequest(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("api_key")
	if s.webhookAPIKey == "" || key != s.webhookAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid API key"))
		return
	}

	reqBody, err := parseInboundSMS(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request: " + err.Error()))
		return
	}

	phone := reqBody.SenderPhone()
	message := reqBody.Message
	if phone == "" || message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing from/contact or message"))
		return
	}

	sharedlogging.Info("notification inbound sms received",
		"phone_hash", sharedlogging.HashIdentifier(phone),
	)

	if err := s.svc.ReceiveSMS(r.Context(), phone, message); err != nil {
		sharedlogging.Error("notification inbound sms processing failed", err, "phone_hash", sharedlogging.HashIdentifier(phone))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("processing failed"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("SMS received successfully"))
}
