package outbound

import "context"

type LiveKitMeetingProvider interface {
	CreateMeetingRoom(ctx context.Context, in LiveKitCreateInput) (*LiveKitCreateResult, error)
	// MintRoomJoinToken issues a join token for an existing room (bridged
	// patient-staff calls) without creating a new room.
	MintRoomJoinToken(ctx context.Context, roomName, identity string) (*LiveKitRoomToken, error)
	// ResolveRoomName maps a LiveKit room SID (e.g. "RM_xxx") to its room name.
	// Join-token grants are keyed by room NAME, but bridged sessions are keyed
	// by the voice session SID, so callers must resolve before minting. If the
	// value is already a room name (or cannot be resolved), it is returned
	// unchanged so callers can fall back safely.
	ResolveRoomName(ctx context.Context, roomSIDOrName string) (string, error)
}

type LiveKitRoomToken struct {
	Token string
	WSURL string
}

type LiveKitCreateInput struct {
	RoomName      string
	Title         string
	EmptyTimeout  uint32
	StartsAtEpoch int64
	DurationSec   int32
}

type LiveKitCreateResult struct {
	RoomName     string
	RoomSID      string
	JoinURL      string
	PatientToken string
	StaffToken   string
}
