package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/inbound"
	sharedTypes "github.com/KoiralaSam/ZorbaHealth/shared/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // restrict to your domains in production
	},
}

// wsConn wraps *websocket.Conn to satisfy inbound.PatientLiveChannel.
type wsConn struct{ *websocket.Conn }

func (w *wsConn) WriteJSON(v any) error { return w.Conn.WriteJSON(v) }
func (w *wsConn) Close() error          { return w.Conn.Close() }

// WebSocketHandler handles GET /ws/location
type WebSocketHandler struct {
	Service inbound.LocationService
	Auth    AuthValidator
}

type AuthValidator interface {
	ExtractPatientID(token string) (string, error)
}

func (h *WebSocketHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = extractBearerToken(r)
	}

	patientID, err := h.Auth.ExtractPatientID(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	ch := &wsConn{conn}
	h.Service.RegisterPatientLiveChannel(r.Context(), patientID, ch)
	log.Printf("patient %s connected to location WS", patientID)

	defer func() {
		h.Service.UnregisterPatientLiveChannel(r.Context(), patientID)
		conn.Close()
		log.Printf("patient %s disconnected from location WS", patientID)
	}()

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if msgType != websocket.TextMessage {
			continue
		}

		var upd sharedTypes.LocationUpdate
		if err := json.Unmarshal(payload, &upd); err != nil {
			log.Printf("invalid location WS payload: %v", err)
			continue
		}

		// If client sends a `type`, enforce it.
		if upd.Type != "" && upd.Type != sharedTypes.LocationUpdateType {
			continue
		}
		if upd.SessionID == "" {
			continue
		}

		loc := models.Location{
			Lat:      upd.Lat,
			Lng:      upd.Lng,
			Accuracy: upd.Accuracy,
			Method:   "gps",
		}
		if err := h.Service.StoreLocation(r.Context(), upd.SessionID, loc); err != nil {
			log.Printf("store location failed (session=%s): %v", upd.SessionID, err)
		}
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
