package livekit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
)

type stubClient struct {
	wsURL string
}

func NewClient() outbound.LiveKitProvider {
	wsURL := strings.TrimSpace(sharedenv.GetString("LIVEKIT_WS_URL", ""))
	if wsURL == "" {
		wsURL = strings.TrimSpace(sharedenv.GetString("LIVEKIT_URL", ""))
	}
	if strings.EqualFold(sharedenv.GetString("LIVEKIT_USE_STUB", ""), "true") {
		return &stubClient{wsURL: wsURL}
	}
	apiKey := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_KEY", ""))
	apiSecret := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_SECRET", ""))
	env := strings.ToLower(strings.TrimSpace(sharedenv.GetString("ENVIRONMENT", "")))
	if apiKey == "" || apiSecret == "" || env == "development" || env == "dev" {
		return &stubClient{wsURL: wsURL}
	}
	// Full LiveKit client can be swapped in; stub is used until production creds are present.
	return &stubClient{wsURL: wsURL}
}

func (s *stubClient) CreateMeetingRoom(_ context.Context, in outbound.LiveKitCreateInput) (*outbound.LiveKitCreateResult, error) {
	roomName := in.RoomName
	if roomName == "" {
		roomName = fmt.Sprintf("stub-appt-%d", time.Now().UnixNano())
	}
	ws := s.wsURL
	if ws == "" {
		ws = "wss://livekit.stub.local"
	}
	return &outbound.LiveKitCreateResult{
		RoomName:     roomName,
		RoomSID:      "stub-" + roomName,
		JoinURL:      ws + "?room=" + roomName + "&stub=1",
		PatientToken: "stub-patient-token",
		StaffToken:   "stub-staff-token",
	}, nil
}
