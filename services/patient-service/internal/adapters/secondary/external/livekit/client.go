package livekit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
	lkauth "github.com/livekit/protocol/auth"
	lklivekit "github.com/livekit/protocol/livekit"
	"google.golang.org/protobuf/types/known/durationpb"
)

const defaultRoomEmptyTimeout uint32 = 60

type Client struct {
	apiKey      string
	apiSecret   string
	wsURL       string
	publicWsURL string
	httpBaseURL string
	httpClient  *http.Client
}

func NewClient() outbound.LiveKitMeetingProvider {
	apiKey := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_KEY", ""))
	apiSecret := strings.TrimSpace(sharedenv.GetString("LIVEKIT_API_SECRET", ""))
	wsURL := strings.TrimSpace(sharedenv.GetString("LIVEKIT_WS_URL", ""))
	if wsURL == "" {
		wsURL = strings.TrimSpace(sharedenv.GetString("LIVEKIT_URL", ""))
	}
	if strings.EqualFold(sharedenv.GetString("LIVEKIT_USE_STUB", ""), "true") {
		return &stubClient{wsURL: wsURL}
	}
	if apiKey == "" || apiSecret == "" {
		env := strings.ToLower(strings.TrimSpace(sharedenv.GetString("ENVIRONMENT", "")))
		if env == "development" || env == "dev" {
			return &stubClient{wsURL: wsURL}
		}
	}
	publicWsURL := strings.TrimSpace(sharedenv.GetString("LIVEKIT_PUBLIC_WS_URL", ""))
	if publicWsURL == "" {
		publicWsURL = wsURL
	}
	httpBaseURL := deriveHTTPBaseURL(wsURL)
	return &Client{
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		wsURL:       wsURL,
		publicWsURL: publicWsURL,
		httpBaseURL: httpBaseURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) CreateMeetingRoom(ctx context.Context, in outbound.LiveKitCreateInput) (*outbound.LiveKitCreateResult, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_API_KEY and LIVEKIT_API_SECRET on patient-service (or LIVEKIT_USE_STUB=true for dev)")
	}
	if c.httpBaseURL == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_URL or LIVEKIT_WS_URL on patient-service")
	}
	token, err := c.adminToken()
	if err != nil {
		return nil, fmt.Errorf("livekit: admin token: %w", err)
	}

	roomClient := lklivekit.NewRoomServiceProtobufClient(c.httpBaseURL, &authClient{
		client: c.httpClient,
		token:  token,
	})

	roomName := in.RoomName
	if roomName == "" {
		roomName = fmt.Sprintf("meeting-%d", time.Now().UnixNano())
	}

	emptyTimeout := in.EmptyTimeout
	if emptyTimeout == 0 {
		emptyTimeout = defaultRoomEmptyTimeout
	}

	created, err := roomClient.CreateRoom(ctx, &lklivekit.CreateRoomRequest{
		Name:           roomName,
		EmptyTimeout:   emptyTimeout,
		Metadata:       in.Title,
		DepartureTimeout: 120,
	})
	if err != nil {
		return nil, fmt.Errorf("livekit: create room: %w", err)
	}

	patientToken, err := c.participantToken(roomName, "patient", true)
	if err != nil {
		return nil, fmt.Errorf("livekit: patient token: %w", err)
	}
	staffToken, err := c.participantToken(roomName, "staff", true)
	if err != nil {
		return nil, fmt.Errorf("livekit: staff token: %w", err)
	}

	return &outbound.LiveKitCreateResult{
		RoomName:     created.Name,
		RoomSID:      created.Sid,
		JoinURL:      c.joinURL(roomName),
		PatientToken: patientToken,
		StaffToken:   staffToken,
	}, nil
}

func (c *Client) MintRoomJoinToken(_ context.Context, roomName, identity string) (*outbound.LiveKitRoomToken, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_API_KEY and LIVEKIT_API_SECRET on patient-service (or LIVEKIT_USE_STUB=true for dev)")
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return nil, fmt.Errorf("livekit: room name is required to mint a join token")
	}
	identity = strings.TrimSpace(identity)
	if identity == "" {
		identity = fmt.Sprintf("staff-%d", time.Now().UnixNano())
	}
	tok := lkauth.NewAccessToken(c.apiKey, c.apiSecret)
	tok.SetIdentity(identity)
	tok.AddGrant(&lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
		CanPublish: func() *bool {
			v := true
			return &v
		}(),
		CanSubscribe: func() *bool {
			v := true
			return &v
		}(),
	})
	jwt, err := tok.SetValidFor(4 * time.Hour).ToJWT()
	if err != nil {
		return nil, fmt.Errorf("livekit: join token: %w", err)
	}
	ws := strings.TrimSpace(c.publicWsURL)
	if ws == "" {
		ws = c.wsURL
	}
	return &outbound.LiveKitRoomToken{Token: jwt, WSURL: ws}, nil
}

// ResolveRoomName maps a LiveKit room SID to its room name via RoomService.
// LiveKit join-token grants are keyed by room NAME, but bridged sessions are
// keyed by the voice session SID ("RM_..."), so a doctor minted against the SID
// would join a brand-new empty room instead of the live SIP call. Any failure
// (no credentials, API unreachable, no match) returns the input unchanged so
// the caller falls back to its prior behaviour.
func (c *Client) ResolveRoomName(ctx context.Context, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if c.apiKey == "" || c.apiSecret == "" || c.httpBaseURL == "" {
		return value, nil
	}
	token, err := c.adminToken()
	if err != nil {
		return value, nil
	}
	roomClient := lklivekit.NewRoomServiceProtobufClient(c.httpBaseURL, &authClient{
		client: c.httpClient,
		token:  token,
	})
	resp, err := roomClient.ListRooms(ctx, &lklivekit.ListRoomsRequest{})
	if err != nil {
		return value, nil
	}
	for _, room := range resp.GetRooms() {
		if room.GetSid() == value {
			return room.GetName(), nil
		}
	}
	return value, nil
}

func (c *Client) adminToken() (string, error) {
	return lkauth.NewAccessToken(c.apiKey, c.apiSecret).
		SetIdentity("livekit-admin").
		AddGrant(&lkauth.VideoGrant{
			RoomCreate: true,
			RoomList:   true,
			RoomAdmin:  true,
		}).
		SetSIPGrant(&lkauth.SIPGrant{Admin: true, Call: true}).
		SetValidFor(5 * time.Minute).
		ToJWT()
}

func (c *Client) DialSIPParticipant(ctx context.Context, in outbound.DialSIPParticipantInput) (*outbound.DialSIPParticipantResult, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_API_KEY and LIVEKIT_API_SECRET on patient-service (or LIVEKIT_USE_STUB=true for dev)")
	}
	if c.httpBaseURL == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_URL or LIVEKIT_WS_URL on patient-service")
	}
	sipTrunkID := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_TRUNK_ID", ""))
	if sipTrunkID == "" {
		return nil, fmt.Errorf("livekit: set LIVEKIT_SIP_TRUNK_ID for staff phone dial-out")
	}
	roomName := strings.TrimSpace(in.RoomName)
	phone := strings.TrimSpace(in.PhoneNumber)
	if roomName == "" || phone == "" {
		return nil, fmt.Errorf("livekit: room name and phone number are required for SIP dial-out")
	}
	identity := strings.TrimSpace(in.ParticipantIdentity)
	if identity == "" {
		identity = "staff-sip-" + normalizeDigits(phone)
	}
	token, err := c.adminToken()
	if err != nil {
		return nil, fmt.Errorf("livekit: admin token: %w", err)
	}
	sipClient := lklivekit.NewSIPProtobufClient(c.httpBaseURL, &authClient{
		client: c.httpClient,
		token:  token,
	})
	req := &lklivekit.CreateSIPParticipantRequest{
		SipTrunkId:          sipTrunkID,
		SipCallTo:           phone,
		RoomName:            roomName,
		ParticipantIdentity: identity,
		ParticipantName:     strings.TrimSpace(in.ParticipantName),
		RingingTimeout:      durationpb.New(45 * time.Second),
		MaxCallDuration:     durationpb.New(30 * time.Minute),
		WaitUntilAnswered:   false,
		PlayDialtone:        true,
	}
	if fromUser := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_FROM_USER", sharedenv.GetString("LIVEKIT_SIP_AUTH_USERNAME", ""))); fromUser != "" {
		req.SipNumber = fromUser
	}
	if callerID := strings.TrimSpace(sharedenv.GetString("LIVEKIT_SIP_CALLER_ID", sharedenv.GetString("VOIPMS_DID", ""))); callerID != "" {
		req.DisplayName = &callerID
	}
	sip, err := sipClient.CreateSIPParticipant(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("livekit: dial staff SIP participant: %w", err)
	}
	return &outbound.DialSIPParticipantResult{
		SIPCallID:           sip.GetSipCallId(),
		ParticipantID:       sip.GetParticipantId(),
		ParticipantIdentity: sip.GetParticipantIdentity(),
	}, nil
}

func (c *Client) participantToken(roomName, identity string, canPublish bool) (string, error) {
	return c.participantTokenValidFor(roomName, identity, canPublish, 48*time.Hour)
}

func (c *Client) RemintMeetingTokens(_ context.Context, roomName string, validFor time.Duration) (string, string, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return "", "", fmt.Errorf("livekit: set LIVEKIT_API_KEY and LIVEKIT_API_SECRET on patient-service (or LIVEKIT_USE_STUB=true for dev)")
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return "", "", fmt.Errorf("livekit: room name is required to remint meeting tokens")
	}
	if validFor < time.Hour {
		validFor = time.Hour
	}
	patientToken, err := c.participantTokenValidFor(roomName, "patient", true, validFor)
	if err != nil {
		return "", "", err
	}
	staffToken, err := c.participantTokenValidFor(roomName, "staff", true, validFor)
	if err != nil {
		return "", "", err
	}
	return patientToken, staffToken, nil
}

func (c *Client) participantTokenValidFor(roomName, identity string, canPublish bool, validFor time.Duration) (string, error) {
	tok := lkauth.NewAccessToken(c.apiKey, c.apiSecret)
	tok.SetIdentity(identity + "-" + roomName)
	tok.AddGrant(&lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
		CanPublish: func() *bool {
			v := canPublish
			return &v
		}(),
		CanSubscribe: func() *bool {
			v := true
			return &v
		}(),
	})
	return tok.SetValidFor(validFor).ToJWT()
}

func (c *Client) joinURL(roomName string) string {
	ws := strings.TrimSpace(c.publicWsURL)
	if ws == "" {
		ws = c.wsURL
	}
	if ws == "" {
		return roomName
	}
	if !strings.HasPrefix(ws, "ws://") && !strings.HasPrefix(ws, "wss://") {
		return roomName
	}
	return ws + "?room=" + url.QueryEscape(roomName)
}

func deriveHTTPBaseURL(wsURL string) string {
	wsURL = strings.TrimSpace(wsURL)
	if wsURL == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(wsURL, "ws://"):
		return "http://" + strings.TrimPrefix(wsURL, "ws://")
	case strings.HasPrefix(wsURL, "wss://"):
		return "https://" + strings.TrimPrefix(wsURL, "wss://")
	default:
		if strings.HasPrefix(wsURL, "http://") || strings.HasPrefix(wsURL, "https://") {
			return wsURL
		}
		return ""
	}
}

type authClient struct {
	client *http.Client
	token  string
}

func (a *authClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+a.token)
	return a.client.Do(req)
}
