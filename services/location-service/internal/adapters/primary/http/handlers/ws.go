package handlers

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"

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
type wsConn struct {
	*websocket.Conn
	clientIP string
}

func (w *wsConn) WriteJSON(v any) error { return w.Conn.WriteJSON(v) }
func (w *wsConn) Close() error          { return w.Conn.Close() }
func (w *wsConn) ClientIP() string      { return w.clientIP }

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

	ch := &wsConn{Conn: conn, clientIP: clientIPFromRequest(r)}
	h.Service.RegisterPatientLiveChannel(r.Context(), patientID, ch)

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

		method := upd.Method
		if method == "" {
			method = "gps"
		}
		loc := models.Location{
			Lat:      upd.Lat,
			Lng:      upd.Lng,
			Accuracy: upd.Accuracy,
			Method:   method,
		}
		if err := h.Service.StoreLocation(r.Context(), upd.SessionID, loc); err != nil {
			log.Printf("store location failed (session=%s): %v", upd.SessionID, err)
			continue
		}
		log.Printf("stored location session=%s method=%s lat=%.5f lng=%.5f", upd.SessionID, method, upd.Lat, upd.Lng)
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}

func clientIPFromRequest(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if host, _, err := net.SplitHostPort(ip); err == nil {
			return host
		}
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
