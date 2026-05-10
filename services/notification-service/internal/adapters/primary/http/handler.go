package http

import (
	"encoding/json"
	"log"
	"net/http"
)

// HandleSMSRequest handles POST /sms (VoIP.ms incoming SMS webhook).
// It validates the api_key query parameter and responds with 200 on success.
func (s *Server) HandleSMSRequest(w http.ResponseWriter, r *http.Request) {
	var reqBody SMSRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body: " + err.Error()))
		return
	}
	defer r.Body.Close()

	key := r.URL.Query().Get("api_key")
	if key != s.webhookAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid API key"))
		return
	}

	log.Printf("SMS request received: %+v\n", reqBody)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("SMS received successfully"))
}
