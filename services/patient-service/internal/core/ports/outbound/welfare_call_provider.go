package outbound

import (
	"context"
	"time"
)

type WelfareCheckCallInput struct {
	RequestID    string
	RunID        string
	RoomName     string
	PatientID    string
	PatientName  string
	PatientPhone string
	ScheduledAt  time.Time
	Timezone     string
	ReasonCode   string
	ReasonDetail string
	PatientToken string
	AgentName    string
}

type WelfareCheckCallResult struct {
	RoomName            string
	RoomSID             string
	DispatchID          string
	SIPCallID           string
	ParticipantID       string
	ParticipantIdentity string
}

type WelfareCheckCallProvider interface {
	StartWelfareCheckCall(ctx context.Context, in WelfareCheckCallInput) (*WelfareCheckCallResult, error)
}
