package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedbridge "github.com/KoiralaSam/ZorbaHealth/shared/bridge"
	translationpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type fakeSessionStore struct {
	session *sharedbridge.Session
	err     error
}

func (f *fakeSessionStore) Load(_ context.Context, _ string) (*sharedbridge.Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

type fakeTranslator struct {
	lastReq        *translationpb.TranslateRequest
	forwardedToken string
	response       *translationpb.TranslateResponse
	err            error
}

func (f *fakeTranslator) Translate(ctx context.Context, in *translationpb.TranslateRequest, _ ...grpc.CallOption) (*translationpb.TranslateResponse, error) {
	f.lastReq = in
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if values := md.Get("x-forwarded-token"); len(values) > 0 {
			f.forwardedToken = values[0]
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func bridgedSession() *sharedbridge.Session {
	return &sharedbridge.Session{
		SessionID:          "room-1",
		Status:             sharedbridge.StatusConnected,
		PatientAccessToken: "patient-jwt",
		StaffAccessToken:   "staff-jwt",
		PatientTranslation: sharedbridge.TranslationPreferences{
			Enabled:      true,
			LanguageMode: "manual",
			LanguageCode: "es",
		},
		StaffTranslation: sharedbridge.TranslationPreferences{
			Enabled:      true,
			LanguageMode: "manual",
			LanguageCode: "en",
		},
	}
}

func postSegment(t *testing.T, srv *server, token string, body relaySegmentRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/internal/interpretation/segment", bytes.NewReader(payload))
	if token != "" {
		req.Header.Set("x-internal-token", token)
	}
	rec := httptest.NewRecorder()
	srv.relaySegment(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) relaySegmentResponse {
	t.Helper()
	var resp relaySegmentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestRelaySegmentRejectsMissingInternalToken(t *testing.T) {
	srv := &server{
		sessions:      &fakeSessionStore{session: bridgedSession()},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "", relaySegmentRequest{SessionID: "room-1", Participant: "patient", Text: "hola"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRelaySegmentTranslatesPatientSegmentForStaff(t *testing.T) {
	translator := &fakeTranslator{response: &translationpb.TranslateResponse{TranslatedText: "hello"}}
	srv := &server{
		sessions:      &fakeSessionStore{session: bridgedSession()},
		translator:    translator,
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{
		SessionID:   "room-1",
		Participant: "patient",
		Text:        "hola",
		SourceLang:  "es",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	resp := decodeResponse(t, rec)
	if resp.Passthrough {
		t.Fatal("expected translated segment, got passthrough")
	}
	if resp.TranslatedText != "hello" || resp.TargetLanguage != "en" {
		t.Fatalf("response = %+v, want translated_text=hello target=en", resp)
	}
	// Patient speech is translated using STAFF prefs and the staff JWT.
	if translator.lastReq.GetTargetLang() != "en" {
		t.Fatalf("target_lang = %q, want en", translator.lastReq.GetTargetLang())
	}
	if translator.forwardedToken != "staff-jwt" {
		t.Fatalf("forwarded token = %q, want staff-jwt", translator.forwardedToken)
	}
}

func TestRelaySegmentPassthroughWhenTargetDisabled(t *testing.T) {
	session := bridgedSession()
	session.StaffTranslation.Enabled = false
	srv := &server{
		sessions:      &fakeSessionStore{session: session},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{SessionID: "room-1", Participant: "patient", Text: "hola"})
	resp := decodeResponse(t, rec)
	if !resp.Passthrough || resp.TranslatedText != "hola" {
		t.Fatalf("response = %+v, want passthrough with original text", resp)
	}
}

func TestRelaySegmentPassthroughWhenLanguageCodeMissing(t *testing.T) {
	session := bridgedSession()
	session.StaffTranslation.LanguageCode = ""
	srv := &server{
		sessions:      &fakeSessionStore{session: session},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{SessionID: "room-1", Participant: "patient", Text: "hola"})
	resp := decodeResponse(t, rec)
	if !resp.Passthrough {
		t.Fatalf("response = %+v, want passthrough when language code empty", resp)
	}
}

func TestRelaySegmentPassthroughWhenSourceMatchesTarget(t *testing.T) {
	srv := &server{
		sessions:      &fakeSessionStore{session: bridgedSession()},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{
		SessionID:   "room-1",
		Participant: "patient",
		Text:        "already english",
		SourceLang:  "en",
	})
	resp := decodeResponse(t, rec)
	if !resp.Passthrough || resp.TargetLanguage != "en" {
		t.Fatalf("response = %+v, want same-language passthrough", resp)
	}
}

func TestRelaySegmentUnavailableWhenTokensMissing(t *testing.T) {
	session := bridgedSession()
	session.PatientAccessToken = ""
	session.StaffAccessToken = ""
	srv := &server{
		sessions:      &fakeSessionStore{session: session},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{SessionID: "room-1", Participant: "patient", Text: "hola"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRelaySegmentGoneWhenSessionEnded(t *testing.T) {
	session := bridgedSession()
	session.Status = sharedbridge.StatusEnded
	srv := &server{
		sessions:      &fakeSessionStore{session: session},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{SessionID: "room-1", Participant: "patient", Text: "hola"})
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410", rec.Code)
	}
}

func TestRelaySegmentNotFound(t *testing.T) {
	srv := &server{
		sessions:      &fakeSessionStore{err: errSessionNotFound},
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{SessionID: "missing", Participant: "staff", Text: "hi"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestRelaySegmentStaffSegmentUsesPatientPreferences(t *testing.T) {
	translator := &fakeTranslator{response: &translationpb.TranslateResponse{TranslatedText: "hola"}}
	srv := &server{
		sessions:      &fakeSessionStore{session: bridgedSession()},
		translator:    translator,
		internalToken: "secret",
	}
	rec := postSegment(t, srv, "secret", relaySegmentRequest{
		SessionID:   "room-1",
		Participant: "staff",
		Text:        "hello",
		SourceLang:  "en",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if translator.lastReq.GetTargetLang() != "es" {
		t.Fatalf("target_lang = %q, want es (patient prefs)", translator.lastReq.GetTargetLang())
	}
	if translator.forwardedToken != "patient-jwt" {
		t.Fatalf("forwarded token = %q, want patient-jwt", translator.forwardedToken)
	}
}
