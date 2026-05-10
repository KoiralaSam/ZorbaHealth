package livekit

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/inbound"
	lklivekit "github.com/livekit/protocol/livekit"
)

type Handler struct {
	Service  inbound.AgentWorkerService
	Verifier interface {
		ReceiveEvent(r *http.Request) (*lklivekit.WebhookEvent, error)
	}

	mu            sync.Mutex
	activeCalls   map[string]context.CancelFunc
	maxCallLength time.Duration
}

type ParticipantMeta struct {
	PatientID string `json:"patientID,omitempty"`
	Language  string `json:"language"`
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if h.Verifier == nil {
		http.Error(w, "webhook verifier is not configured", http.StatusInternalServerError)
		return
	}

	event, err := h.Verifier.ReceiveEvent(r)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	roomName := ""
	if room := event.GetRoom(); room != nil {
		roomName = room.GetName()
	}

	participantIdentity := ""
	participantMetadata := ""
	if participant := event.GetParticipant(); participant != nil {
		participantIdentity = participant.GetIdentity()
		participantMetadata = participant.GetMetadata()
	}

	log.Printf(
		"livekit webhook received event=%q room=%q participant=%q metadata=%q",
		event.GetEvent(),
		roomName,
		participantIdentity,
		participantMetadata,
	)

	participant := event.GetParticipant()
	if participant == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	if strings.HasPrefix(participant.GetIdentity(), "agent-worker-") {
		w.WriteHeader(http.StatusOK)
		return
	}

	// We only start/stop sessions based on join/leave.
	switch event.GetEvent() {
	case "participant_joined":
		// continue below
	case "participant_left":
		h.cancelActiveCall(event)
		w.WriteHeader(http.StatusOK)
		return
	default:
		w.WriteHeader(http.StatusOK)
		return
	}

	var meta ParticipantMeta
	if participant.GetMetadata() != "" {
		if err := json.Unmarshal([]byte(participant.GetMetadata()), &meta); err != nil {
			http.Error(w, "invalid participant metadata", http.StatusBadRequest)
			return
		}
	}
	if meta.Language == "" {
		meta.Language = "en"
	}

	callerPhone := extractCallerPhone(participantIdentity)
	// Prevent spurious sessions when SIP identities are extensions (e.g. sip_1001).
	// For inbound calls we expect at least a 10-digit phone number.
	if len(callerPhone) < 10 {
		log.Printf("livekit ignoring participant_joined with short caller identity=%q digits=%q", participantIdentity, callerPhone)
		w.WriteHeader(http.StatusOK)
		return
	}

	go func() {
		room := event.GetRoom()
		if room == nil {
			log.Printf("livekit webhook missing room on participant_joined event")
			return
		}
		ctx, cancel := h.registerActiveCall(room.GetSid(), participant.GetIdentity())
		if cancel == nil {
			// already running for this room+participant
			return
		}
		defer func() {
			cancel()
			h.unregisterActiveCall(room.GetSid(), participant.GetIdentity())
		}()

		err := h.Service.StartSession(ctx, models.SessionStart{
			RoomName:      room.GetName(),
			SessionID:     room.GetSid(),
			Language:      meta.Language,
			CallerPhone:   callerPhone,
			PatientIDHint: meta.PatientID,
		})
		if err != nil {
			log.Printf("agent-worker session failed room=%s caller=%s patient_hint=%s session=%s: %v", room.GetName(), callerPhone, meta.PatientID, room.GetSid(), err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) registerActiveCall(roomSID, participantIdentity string) (context.Context, context.CancelFunc) {
	roomSID = strings.TrimSpace(roomSID)
	participantIdentity = strings.TrimSpace(participantIdentity)
	if roomSID == "" || participantIdentity == "" {
		return context.Background(), nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.activeCalls == nil {
		h.activeCalls = make(map[string]context.CancelFunc)
	}
	if h.maxCallLength == 0 {
		h.maxCallLength = 2 * time.Hour
	}

	key := roomSID + "|" + participantIdentity
	if _, ok := h.activeCalls[key]; ok {
		log.Printf("livekit session already active room_sid=%s participant=%s", roomSID, participantIdentity)
		return context.Background(), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.maxCallLength)
	h.activeCalls[key] = cancel
	return ctx, cancel
}

func (h *Handler) unregisterActiveCall(roomSID, participantIdentity string) {
	roomSID = strings.TrimSpace(roomSID)
	participantIdentity = strings.TrimSpace(participantIdentity)
	if roomSID == "" || participantIdentity == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.activeCalls == nil {
		return
	}
	delete(h.activeCalls, roomSID+"|"+participantIdentity)
}

func (h *Handler) cancelActiveCall(event *lklivekit.WebhookEvent) {
	room := event.GetRoom()
	participant := event.GetParticipant()
	if room == nil || participant == nil {
		return
	}
	roomSID := strings.TrimSpace(room.GetSid())
	participantIdentity := strings.TrimSpace(participant.GetIdentity())
	if roomSID == "" || participantIdentity == "" {
		return
	}
	key := roomSID + "|" + participantIdentity

	h.mu.Lock()
	cancel := func() context.CancelFunc {
		if h.activeCalls == nil {
			return nil
		}
		return h.activeCalls[key]
	}()
	h.mu.Unlock()

	if cancel != nil {
		log.Printf("livekit canceling active session room_sid=%s participant=%s", roomSID, participantIdentity)
		cancel()
	}
}

func extractCallerPhone(identity string) string {
	identity = strings.TrimSpace(identity)
	identity = strings.TrimPrefix(identity, "sip_")

	var digits strings.Builder
	for _, r := range identity {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	return digits.String()
}
