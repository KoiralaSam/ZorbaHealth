package livekit

import (
	"context"
	"fmt"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
)

// stubClient satisfies scheduling accept/reschedule without calling a LiveKit server.
type stubClient struct {
	wsURL string
}

func (s *stubClient) CreateMeetingRoom(_ context.Context, in outbound.LiveKitCreateInput) (*outbound.LiveKitCreateResult, error) {
	roomName := in.RoomName
	if roomName == "" {
		roomName = fmt.Sprintf("stub-meeting-%d", time.Now().UnixNano())
	}
	ws := s.wsURL
	if ws == "" {
		ws = "wss://livekit.stub.local"
	}
	join := ws + "?room=" + roomName + "&stub=1"
	return &outbound.LiveKitCreateResult{
		RoomName:     roomName,
		RoomSID:      "stub-" + roomName,
		JoinURL:      join,
		PatientToken: "stub-patient-token",
		StaffToken:   "stub-staff-token",
	}, nil
}

func (s *stubClient) MintRoomJoinToken(_ context.Context, roomName, identity string) (*outbound.LiveKitRoomToken, error) {
	ws := s.wsURL
	if ws == "" {
		ws = "wss://livekit.stub.local"
	}
	if identity == "" {
		identity = "staff"
	}
	return &outbound.LiveKitRoomToken{
		Token: fmt.Sprintf("stub-join-%s-%s", identity, roomName),
		WSURL: ws,
	}, nil
}

// ResolveRoomName is a no-op for the stub: dev rooms are addressed by name.
func (s *stubClient) ResolveRoomName(_ context.Context, value string) (string, error) {
	return value, nil
}
