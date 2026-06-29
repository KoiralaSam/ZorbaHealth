package meetingjoin

import (
	"net/url"
	"strings"
)

// LiveKitServerURL strips ?room= (and other query) from a stored join_url base.
func LiveKitServerURL(storedJoinURL string) string {
	storedJoinURL = strings.TrimSpace(storedJoinURL)
	if storedJoinURL == "" {
		return ""
	}
	u, err := url.Parse(storedJoinURL)
	if err != nil {
		return storedJoinURL
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

// RoomFromJoinURL reads the room query param from a legacy stored join_url.
func RoomFromJoinURL(storedJoinURL string) string {
	u, err := url.Parse(strings.TrimSpace(storedJoinURL))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(u.Query().Get("room"))
}

// WithParticipantToken appends token and role for email / deep-link style URLs.
func WithParticipantToken(storedJoinURL, token, role string) string {
	storedJoinURL = strings.TrimSpace(storedJoinURL)
	token = strings.TrimSpace(token)
	if storedJoinURL == "" || token == "" {
		return storedJoinURL
	}
	sep := "?"
	if strings.Contains(storedJoinURL, "?") {
		sep = "&"
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "participant"
	}
	return storedJoinURL + sep + "token=" + url.QueryEscape(token) + "&role=" + url.QueryEscape(role)
}

// WebAppJoinURL links to the Next.js join page (HTTPS-safe; token stays in query for one-time join).
func WebAppJoinURL(webBase, serverWS, room, token string) string {
	webBase = strings.TrimRight(strings.TrimSpace(webBase), "/")
	serverWS = strings.TrimSpace(serverWS)
	room = strings.TrimSpace(room)
	token = strings.TrimSpace(token)
	if webBase == "" || serverWS == "" || room == "" || token == "" {
		return ""
	}
	v := url.Values{}
	v.Set("server", serverWS)
	v.Set("room", room)
	v.Set("token", token)
	return webBase + "/meeting/join?" + v.Encode()
}
