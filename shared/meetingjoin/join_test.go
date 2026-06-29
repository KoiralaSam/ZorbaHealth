package meetingjoin

import (
	"strings"
	"testing"
)

func TestLiveKitServerURL(t *testing.T) {
	got := LiveKitServerURL("ws://10.0.0.1:7880?room=meeting-abc")
	if got != "ws://10.0.0.1:7880" {
		t.Fatalf("server url: %q", got)
	}
}

func TestWebAppJoinURL(t *testing.T) {
	got := WebAppJoinURL("https://zorbahealth.dev", "wss://lk.example.com", "meeting-1", "jwt-here")
	if got == "" || !strings.Contains(got, "/meeting/join?") || !strings.Contains(got, "token=") {
		t.Fatalf("join url: %q", got)
	}
}
