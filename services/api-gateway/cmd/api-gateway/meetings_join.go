package main

import (
	"os"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/shared/meetingjoin"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
)

type meetingJoinRole string

const (
	meetingJoinRolePatient meetingJoinRole = "patient"
	meetingJoinRoleStaff   meetingJoinRole = "staff"
)

func meetingJoinURLForRole(m *schedpb.ScheduledMeeting, role meetingJoinRole) string {
	if m == nil {
		return ""
	}
	room := strings.TrimSpace(m.GetLivekitRoomName())
	if room == "" {
		room = meetingjoin.RoomFromJoinURL(m.GetJoinUrl())
	}
	var token string
	switch role {
	case meetingJoinRolePatient:
		token = m.GetPatientToken()
	case meetingJoinRoleStaff:
		token = m.GetStaffToken()
	}
	if token == "" {
		return strings.TrimSpace(m.GetJoinUrl())
	}

	serverWS := meetingjoin.LiveKitServerURL(m.GetJoinUrl())
	if pub := strings.TrimSpace(os.Getenv("LIVEKIT_PUBLIC_WS_URL")); pub != "" {
		serverWS = pub
	}

	webBase := strings.TrimSpace(os.Getenv("PUBLIC_WEB_BASE_URL"))
	if web := meetingjoin.WebAppJoinURL(webBase, serverWS, room, token); web != "" {
		return web
	}
	return meetingjoin.WithParticipantToken(m.GetJoinUrl(), token, string(role))
}
