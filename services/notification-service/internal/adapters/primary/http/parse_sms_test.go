package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseInboundSMS_GETQuery(t *testing.T) {
	q := url.Values{}
	q.Set("id", "1")
	q.Set("from", "5551234567")
	q.Set("did", "3185162690")
	q.Set("message", "123456")
	req := httptest.NewRequest(http.MethodGet, "/sms?"+q.Encode(), nil)

	got, err := parseInboundSMS(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SenderPhone() != "5551234567" || got.Message != "123456" {
		t.Fatalf("got=%+v", got)
	}
}

func TestParseInboundSMS_VoipmsContactField(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/sms?contact=5559876543&message=hi", nil)
	got, err := parseInboundSMS(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SenderPhone() != "5559876543" || got.Message != "hi" {
		t.Fatalf("got=%+v", got)
	}
}

func TestParseInboundSMS_JSONPost(t *testing.T) {
	body := strings.NewReader(`{"from":"5551112222","message":"otp"}`)
	req := httptest.NewRequest(http.MethodPost, "/sms", body)
	req.Header.Set("Content-Type", "application/json")

	got, err := parseInboundSMS(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.SenderPhone() != "5551112222" || got.Message != "otp" {
		t.Fatalf("got=%+v", got)
	}
}
