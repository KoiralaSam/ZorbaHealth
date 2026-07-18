package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	sharedbridge "github.com/KoiralaSam/ZorbaHealth/shared/bridge"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	translationpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errSessionNotFound = errors.New("bridged session not found")

// sessionStore abstracts the Redis lookup so the relay handler is testable.
type sessionStore interface {
	Load(ctx context.Context, sessionID string) (*sharedbridge.Session, error)
}

type server struct {
	sessions      sessionStore
	translator    translationpb.TranslationServiceClient
	audit         auditpb.AuditServiceClient
	internalToken string
}

type relaySegmentRequest struct {
	SessionID   string `json:"session_id"`
	Participant string `json:"participant"`
	Text        string `json:"text"`
	SourceLang  string `json:"source_lang,omitempty"`
	// TargetHint is the listening party's language as observed live on the
	// call (e.g. the SIP-detected patient language). It is used only when the
	// target party enabled auto-mode translation without an explicit
	// LanguageCode, so live detection drives the target instead of passthrough.
	TargetHint string `json:"target_hint,omitempty"`
}

type relaySegmentResponse struct {
	SessionID      string `json:"session_id"`
	TargetLanguage string `json:"target_language,omitempty"`
	TranslatedText string `json:"translated_text"`
	Passthrough    bool   `json:"passthrough"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "interpretation-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	shutdown, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     env.GetString("REDIS_ADDR", "redis:6379"),
		Password: env.GetString("REDIS_PASSWORD", ""),
		DB:       0,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}

	translationConn, err := grpcclient.Dial(env.GetString("TRANSLATION_SERVICE_GRPC_ADDR", "translation-service:50057"))
	if err != nil {
		log.Fatalf("translation-service dial: %v", err)
	}
	defer translationConn.Close()

	auditConn, err := grpcclient.Dial(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Fatalf("audit-service dial: %v", err)
	}
	defer auditConn.Close()

	internalToken := env.GetString("INTERNAL_SERVICE_SECRET", "")
	if internalToken == "" {
		log.Printf("WARNING: INTERNAL_SERVICE_SECRET is empty; /internal/interpretation/segment accepts unauthenticated callers")
	}

	srv := &server{
		sessions:      &redisSessionStore{client: redisClient},
		translator:    translationpb.NewTranslationServiceClient(translationConn),
		audit:         auditpb.NewAuditServiceClient(auditConn),
		internalToken: internalToken,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "interpretation-service"})
	})
	mux.HandleFunc("POST /internal/interpretation/segment", srv.relaySegment)

	httpAddr := env.GetString("INTERPRETATION_SERVICE_HTTP_ADDR", ":8095")
	log.Printf("interpretation-service listening on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, otelhttp.NewHandler(mux, "interpretation-service")); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// relaySegment translates one final STT utterance for the counterparty.
//
// Passthrough rules (documented in services/interpretation-service/README.md):
//  1. The TARGET party's preferences gate translation: a segment spoken by the
//     patient is translated using the STAFF preferences and vice versa.
//  2. Passthrough (no translation) when the target has Enabled=false or an
//     empty LanguageCode.
//  3. Passthrough when the segment's source_lang equals the target language.
//  4. "auto" language mode applies to the segment source_lang (empty
//     source_lang lets the translation provider auto-detect); the target
//     LanguageCode must always be explicit for translation to happen.
func (s *server) relaySegment(w http.ResponseWriter, r *http.Request) {
	if s.internalToken != "" && r.Header.Get("x-internal-token") != s.internalToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req relaySegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	session, err := s.sessions.Load(r.Context(), req.SessionID)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.Status == sharedbridge.StatusEnded {
		http.Error(w, "bridged session has ended", http.StatusGone)
		return
	}

	targetPrefs, forwardedToken := targetPreferences(session, strings.TrimSpace(strings.ToLower(req.Participant)))
	targetLang := resolveTargetLanguage(targetPrefs, req.TargetHint)
	if !targetPrefs.Enabled || targetLang == "" {
		s.appendSegmentAudit(r.Context(), req, "", true)
		writeJSON(w, http.StatusOK, relaySegmentResponse{
			SessionID:      req.SessionID,
			TranslatedText: req.Text,
			Passthrough:    true,
		})
		return
	}
	if strings.TrimSpace(forwardedToken) == "" {
		http.Error(w, "missing bridged actor token", http.StatusServiceUnavailable)
		return
	}
	if source := strings.TrimSpace(strings.ToLower(req.SourceLang)); source != "" && source == targetLang {
		s.appendSegmentAudit(r.Context(), req, targetLang, true)
		writeJSON(w, http.StatusOK, relaySegmentResponse{
			SessionID:      req.SessionID,
			TargetLanguage: targetLang,
			TranslatedText: req.Text,
			Passthrough:    true,
		})
		return
	}

	ctx := metadata.NewOutgoingContext(r.Context(), metadata.Pairs(
		"x-internal-token", s.internalToken,
		"x-forwarded-token", forwardedToken,
	))
	resp, err := s.translator.Translate(ctx, &translationpb.TranslateRequest{
		Text:       req.Text,
		SourceLang: strings.TrimSpace(strings.ToLower(req.SourceLang)),
		TargetLang: targetLang,
	})
	if err != nil {
		http.Error(w, "translation failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	s.appendSegmentAudit(r.Context(), req, targetLang, false)
	writeJSON(w, http.StatusOK, relaySegmentResponse{
		SessionID:      req.SessionID,
		TargetLanguage: targetLang,
		TranslatedText: resp.GetTranslatedText(),
		Passthrough:    false,
	})
}

type redisSessionStore struct {
	client *redis.Client
}

func (r *redisSessionStore) Load(ctx context.Context, sessionID string) (*sharedbridge.Session, error) {
	raw, err := r.client.Get(ctx, sharedbridge.Key(strings.TrimSpace(sessionID))).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errSessionNotFound
		}
		return nil, err
	}
	var session sharedbridge.Session
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// resolveTargetLanguage picks the language to translate INTO. An explicit
// LanguageCode always wins. When the target enabled auto-mode translation
// without choosing a code, the live target hint (e.g. the SIP-detected patient
// language) is used so the doctor->patient direction interprets instead of
// falling back to passthrough. Manual mode keeps its explicit-only contract.
func resolveTargetLanguage(prefs sharedbridge.TranslationPreferences, hint string) string {
	if code := strings.TrimSpace(strings.ToLower(prefs.LanguageCode)); code != "" {
		return code
	}
	mode := strings.TrimSpace(strings.ToLower(prefs.LanguageMode))
	if mode != "" && mode != "auto" {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(hint))
}

// targetPreferences returns the counterparty's preferences and the JWT used
// to authorize the translation call on their behalf.
func targetPreferences(session *sharedbridge.Session, participant string) (sharedbridge.TranslationPreferences, string) {
	switch participant {
	case sharedbridge.ParticipantPatient:
		return session.StaffTranslation, coalesce(session.StaffAccessToken, session.PatientAccessToken)
	case sharedbridge.ParticipantStaff:
		return session.PatientTranslation, coalesce(session.PatientAccessToken, session.StaffAccessToken)
	default:
		return sharedbridge.TranslationPreferences{}, ""
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *server) appendSegmentAudit(ctx context.Context, req relaySegmentRequest, targetLanguage string, passthrough bool) {
	if s.audit == nil {
		return
	}
	meta, _ := structpb.NewStruct(map[string]any{
		"session_id":      strings.TrimSpace(req.SessionID),
		"participant":     strings.TrimSpace(strings.ToLower(req.Participant)),
		"source_language": strings.TrimSpace(strings.ToLower(req.SourceLang)),
		"target_language": strings.TrimSpace(strings.ToLower(targetLanguage)),
		"passthrough":     passthrough,
		"character_count": len([]rune(req.Text)),
	})
	_, _ = s.audit.AppendAuditEvent(metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-internal-token", s.internalToken,
	)), &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     "INTERPRETATION_SEGMENT_PROCESSED",
			ActorType:     "system_service",
			ActorId:       "interpretation-service",
			ServiceName:   "interpretation-service",
			Timestamp:     timestamppb.New(time.Now().UTC()),
			SuccessStatus: true,
			Metadata:      meta,
		},
	})
}
