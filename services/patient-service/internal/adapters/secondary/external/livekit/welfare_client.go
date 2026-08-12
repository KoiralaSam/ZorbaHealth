package livekit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
	lkauth "github.com/livekit/protocol/auth"
	lklivekit "github.com/livekit/protocol/livekit"
	"google.golang.org/protobuf/types/known/durationpb"
)

// WelfareClient dispatches scheduled welfare-check SIP calls via LiveKit.
// Distinct from Client (meeting rooms) so SIP/agent grants stay scoped here.
type WelfareClient struct {
	apiKey      string
	apiSecret   string
	wsURL       string
	httpBaseURL string
	sipTrunkID  string
	agentName   string
	httpClient  *http.Client
}

func NewWelfareCheckCallProvider() outbound.WelfareCheckCallProvider {
	if sharedenv.GetBool("LIVEKIT_USE_STUB", false) {
		return &stubWelfareCallProvider{}
	}
	apiKey := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_KEY", ""))
	apiSecret := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_SECRET", ""))
	wsURL := strings.TrimSpace(sharedenv.GetString("LIVEKIT_WS_URL", ""))
	if wsURL == "" {
		wsURL = strings.TrimSpace(sharedenv.GetString("LIVEKIT_URL", ""))
	}
	sipTrunkID := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_TRUNK_ID", ""))
	return &WelfareClient{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		wsURL:       wsURL,
		httpBaseURL: deriveHTTPBaseURL(wsURL),
		sipTrunkID:  sipTrunkID,
		agentName:   strings.TrimSpace(sharedenv.GetString("LIVEKIT_AGENT_NAME", "zorba-health-voice")),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *WelfareClient) StartWelfareCheckCall(ctx context.Context, in outbound.WelfareCheckCallInput) (*outbound.WelfareCheckCallResult, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_API_KEY and LIVEKIT_API_SECRET")
	}
	if c.httpBaseURL == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_URL or LIVEKIT_WS_URL")
	}
	if c.sipTrunkID == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_SIP_TRUNK_ID")
	}
	roomName := strings.TrimSpace(in.RoomName)
	if roomName == "" {
		if strings.TrimSpace(in.RunID) == "" {
			return nil, fmt.Errorf("livekit: run_id is required for stable room identity")
		}
		roomName = "welfare-check-" + strings.TrimSpace(in.RunID)
	}
	token, err := c.welfareAdminToken(roomName)
	if err != nil {
		return nil, err
	}
	admin := &authClient{client: c.httpClient, token: token}

	roomClient := lklivekit.NewRoomServiceProtobufClient(c.httpBaseURL, admin)
	room, err := roomClient.CreateRoom(ctx, &lklivekit.CreateRoomRequest{
		Name:             roomName,
		EmptyTimeout:     180,
		DepartureTimeout: 120,
		Metadata:         "scheduled welfare check",
	})
	if err != nil {
		// Room may already exist from a prior partial attempt; continue with known name.
		room = &lklivekit.Room{Name: roomName}
	}

	publicMetadata, err := json.Marshal(map[string]any{
		"type":          "welfare_check",
		"request_id":    in.RequestID,
		"run_id":        in.RunID,
		"patient_id":    in.PatientID,
		"patient_name":  in.PatientName,
		"reason":        in.ReasonCode,
		"reason_code":   in.ReasonCode,
		"reason_detail": in.ReasonDetail,
		"scheduled_at":  in.ScheduledAt.UTC().Format(time.RFC3339),
		"timezone":      in.Timezone,
	})
	if err != nil {
		return nil, err
	}
	agentMetadata, err := json.Marshal(map[string]any{
		"type":          "welfare_check",
		"request_id":    in.RequestID,
		"run_id":        in.RunID,
		"patient_id":    in.PatientID,
		"patient_name":  in.PatientName,
		"reason":        in.ReasonCode,
		"reason_code":   in.ReasonCode,
		"reason_detail": in.ReasonDetail,
		"scheduled_at":  in.ScheduledAt.UTC().Format(time.RFC3339),
		"timezone":      in.Timezone,
		"patient_token": in.PatientToken,
	})
	if err != nil {
		return nil, err
	}
	agentName := strings.TrimSpace(in.AgentName)
	if agentName == "" {
		agentName = c.agentName
	}
	dispatchClient := lklivekit.NewAgentDispatchServiceProtobufClient(c.httpBaseURL, admin)
	dispatch, err := dispatchClient.CreateDispatch(ctx, &lklivekit.CreateAgentDispatchRequest{
		AgentName: agentName,
		Room:      roomName,
		Metadata:  string(agentMetadata),
	})
	if err != nil {
		return nil, fmt.Errorf("livekit: dispatch welfare agent: %w", err)
	}

	sipClient := lklivekit.NewSIPProtobufClient(c.httpBaseURL, admin)
	identity := "sip_" + normalizeDigits(in.PatientPhone)
	req := &lklivekit.CreateSIPParticipantRequest{
		SipTrunkId:          c.sipTrunkID,
		SipCallTo:           in.PatientPhone,
		RoomName:            roomName,
		ParticipantIdentity: identity,
		ParticipantName:     in.PatientName,
		ParticipantMetadata: string(publicMetadata),
		RingingTimeout:      durationpb.New(45 * time.Second),
		MaxCallDuration:     durationpb.New(15 * time.Minute),
		// Wait so SIP declines / no-answer surface as dial errors and enter the
		// welfare retry path instead of being marked dispatched prematurely.
		WaitUntilAnswered: true,
		PlayDialtone:      true,
	}
	// Voip.ms requires From URI user = SIP account (e.g. 527931), not the DID.
	// Do not set req.Trunk here: a partial Trunk override clears the stored trunk
	// hostname and yields "no outbound hostname specified".
	if fromUser := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_FROM_USER", sharedenv.GetString("LIVEKIT_SIP_AUTH_USERNAME", ""))); fromUser != "" {
		req.SipNumber = fromUser
	}
	// Optional PSTN Caller ID display (DID). Separate from From URI user.
	if callerID := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_CALLER_ID", sharedenv.GetString("VOIPMS_DID", ""))); callerID != "" {
		req.DisplayName = &callerID
	}
	sip, err := sipClient.CreateSIPParticipant(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("livekit: start outbound SIP call: %w", err)
	}

	return &outbound.WelfareCheckCallResult{
		RoomName:            firstNonEmpty(room.GetName(), roomName),
		RoomSID:             room.GetSid(),
		DispatchID:          dispatch.GetId(),
		SIPCallID:           sip.GetSipCallId(),
		ParticipantID:       sip.GetParticipantId(),
		ParticipantIdentity: sip.GetParticipantIdentity(),
	}, nil
}

func (c *WelfareClient) welfareAdminToken(roomName string) (string, error) {
	// LiveKit authorizes AgentDispatch via room-scoped roomAdmin (same as `lk dispatch create`),
	// not a global agent.admin grant alone.
	return lkauth.NewAccessToken(c.apiKey, c.apiSecret).
		SetIdentity("welfare-check-dispatcher").
		SetVideoGrant(&lkauth.VideoGrant{
			RoomCreate: true,
			RoomList:   true,
			RoomAdmin:  true,
			Room:       roomName,
		}).
		SetSIPGrant(&lkauth.SIPGrant{Admin: true, Call: true}).
		SetAgentGrant(&lkauth.AgentGrant{Admin: true}).
		SetValidFor(5 * time.Minute).
		ToJWT()
}

func normalizeDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type stubWelfareCallProvider struct{}

func (s *stubWelfareCallProvider) StartWelfareCheckCall(_ context.Context, in outbound.WelfareCheckCallInput) (*outbound.WelfareCheckCallResult, error) {
	room := strings.TrimSpace(in.RoomName)
	if room == "" {
		room = "welfare-check-" + in.RunID
	}
	return &outbound.WelfareCheckCallResult{
		RoomName:            room,
		RoomSID:             "stub-" + in.RunID,
		DispatchID:          "dispatch-" + in.RunID,
		SIPCallID:           "sip-" + in.RunID,
		ParticipantID:       "participant-" + in.RunID,
		ParticipantIdentity: "sip_" + normalizeDigits(in.PatientPhone),
	}, nil
}
