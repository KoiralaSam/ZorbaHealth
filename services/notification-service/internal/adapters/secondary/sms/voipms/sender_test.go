package voipms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSMS_APIErrorIncludesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("method") != "sendSMS" {
			t.Fatalf("method=%q", r.URL.Query().Get("method"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"invalid_did","message":"The DID is not valid"}`))
	}))
	t.Cleanup(srv.Close)

	s := NewSender(srv.URL, "user@example.com", "secret", "3185162690")
	err := s.SendSMS(context.Background(), "+15551234567", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "invalid_did") || !strings.Contains(msg, "not valid") {
		t.Fatalf("error=%q", msg)
	}
}

func TestSendSMS_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","sms":"12345"}`))
	}))
	t.Cleanup(srv.Close)

	s := NewSender(srv.URL, "user@example.com", "secret", "3185162690")
	if err := s.SendSMS(context.Background(), "5551234567", "code 123456"); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"+1 (555) 123-4567", "5551234567"},
		{"15551234567", "5551234567"},
		{"5551234567", "5551234567"},
	}
	for _, tt := range tests {
		if got := normalizePhone(tt.in); got != tt.want {
			t.Fatalf("normalizePhone(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}
