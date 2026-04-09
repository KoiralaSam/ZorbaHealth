package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
)

const (
	sttSampleRate = 24000
	sttChannels   = 1
)

type AgentWorkerService struct {
	tokenIssuer     outbound.SessionTokenIssuer
	patientIdentity outbound.PatientIdentityClient
	health          outbound.HealthRecordsClient
	stt             outbound.Transcriber
	llm             outbound.ChatModel
	tts             outbound.Synthesizer
	mcp             outbound.ToolCaller
	roomAudio       outbound.RoomAudioGateway
}

var _ inbound.AgentWorkerService = (*AgentWorkerService)(nil)

func NewAgentWorkerService(
	tokenIssuer outbound.SessionTokenIssuer,
	patientIdentity outbound.PatientIdentityClient,
	health outbound.HealthRecordsClient,
	stt outbound.Transcriber,
	llm outbound.ChatModel,
	tts outbound.Synthesizer,
	mcp outbound.ToolCaller,
	roomAudio outbound.RoomAudioGateway,
) inbound.AgentWorkerService {
	return &AgentWorkerService{
		tokenIssuer:     tokenIssuer,
		patientIdentity: patientIdentity,
		health:          health,
		stt:             stt,
		llm:             llm,
		tts:             tts,
		mcp:             mcp,
		roomAudio:       roomAudio,
	}
}

func (s *AgentWorkerService) StartSession(ctx context.Context, start models.SessionStart) error {
	if strings.TrimSpace(start.RoomName) == "" {
		return errors.New("room name is required")
	}
	if strings.TrimSpace(start.SessionID) == "" {
		return errors.New("session ID is required")
	}
	if strings.TrimSpace(start.Language) == "" {
		start.Language = "en"
	}

	state := &models.SessionState{
		RoomName:    start.RoomName,
		SessionID:   start.SessionID,
		Language:    start.Language,
		CallerPhone: start.CallerPhone,
	}
	systemPrompt := s.llm.BuildUnauthenticatedSystemPrompt(start.Language, start.CallerPhone)
	allPublicTools := s.llm.BuildPublicToolDefinitions()
	limitedPublicTools := filterToolsByName(allPublicTools, map[string]bool{
		"translate":             true,
		"get_location":          true,
		"find_nearest_hospital": true,
	})
	tools := limitedPublicTools
	messages := make([]models.Message, 0, 16)

	log.Printf(
		"agent-worker session start room=%s session=%s caller=%s patient_hint=%s",
		start.RoomName,
		start.SessionID,
		start.CallerPhone,
		start.PatientIDHint,
	)

	defer defaultUtteranceCollector.delete(start.SessionID)

	// Mint a restricted, session-scoped token immediately so the caller can use
	// "public" tools (translate/location/hospital) without being forced into registration.
	// We use the session ID as the patientID placeholder until verification completes.
	{
		token, err := s.tokenIssuer.MintSessionToken(
			"session:"+start.SessionID,
			start.SessionID,
			[]string{"location:read", "emergency:write"},
		)
		if err != nil {
			return fmt.Errorf("mint provisional session token: %w", err)
		}
		state.Token = token
	}

	// Speak an immediate greeting so the caller isn't met with silence while waiting
	// for their first utterance. If TTS fails, continue the session anyway.
	{
		greeting := "Hi, this is Zorba Health. How can I help today? " +
			"I can answer general health questions and help in emergencies. " +
			"If you want me to access your personal medical records, I will first verify your phone number."
		if audio, err := s.tts.Speak(greeting, start.Language); err != nil {
			log.Printf("agent-worker greeting TTS failed room=%s session=%s: %v", start.RoomName, start.SessionID, err)
		} else if len(audio) > 0 {
			markAssistantSpeaking(state, greeting, audio)
			if err := s.roomAudio.PublishAudio(ctx, start.RoomName, audio); err != nil {
				log.Printf("agent-worker greeting publish failed room=%s session=%s: %v", start.RoomName, start.SessionID, err)
			} else {
				messages = append(messages, models.Message{Role: "assistant", Content: greeting})
			}
		}
	}

	for {
		audioChunk, err := s.roomAudio.NextAudioChunk(ctx, start.RoomName)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read room audio: %w", err)
		}
		if len(audioChunk) == 0 {
			continue
		}

		// While the assistant is speaking, don't process inbound audio. This avoids
		// STT echo triggers and prevents the agent from interrupting itself.
		if !state.SpeakingUntil.IsZero() && time.Now().Before(state.SpeakingUntil) {
			continue
		}

		// Accumulate caller speech into a single utterance and only transcribe once a silence chunk arrives.
		text, ok, err := s.collectUtteranceText(ctx, state, audioChunk, start.Language)
		if err != nil {
			return fmt.Errorf("collect utterance: %w", err)
		}
		if !ok {
			continue
		}

		messages = append(messages, models.Message{
			Role:    "user",
			Content: text,
		})

		// Only allow verification/registration tools when the user asked for something
		// that likely requires a real patient identity (records, meds, results, etc).
		if state.Patient != nil {
			state.IdentityGateEnabled = false
			state.IdentityGateSticky = false
		} else {
			if requiresIdentity(text) {
				// Once the caller expresses intent to access their profile/records,
				// keep verification tools available across follow-up turns where they
				// only provide name/DOB/OTP.
				state.IdentityGateSticky = true
			}
			state.IdentityGateEnabled = state.IdentityGateSticky || state.VerificationMode != "" || state.RegistrationToken != ""
		}
		if state.IdentityGateEnabled {
			tools = allPublicTools
		} else {
			tools = limitedPublicTools
		}

		if state.Patient == nil {
			if err := s.discoverPatientCandidate(ctx, state); err != nil {
				return fmt.Errorf("discover patient candidate: %w", err)
			}
		}

		reply, updatedMessages, err := s.runLLMLoop(ctx, systemPrompt, tools, state, messages)
		if err != nil {
			return fmt.Errorf("llm loop: %w", err)
		}
		messages = updatedMessages

		audioReply, err := s.tts.Speak(reply, start.Language)
		if err != nil {
			return fmt.Errorf("synthesize speech: %w", err)
		}
		markAssistantSpeaking(state, reply, audioReply)
		if err := s.roomAudio.PublishAudio(ctx, start.RoomName, audioReply); err != nil {
			return fmt.Errorf("publish audio: %w", err)
		}

		if state.Patient != nil && state.Token != "" {
			grpcCtx := grpcclient.WithForwardedToken(ctx, state.Token)
			if err := s.health.SaveConversationTurn(grpcCtx, state.Patient.PatientID, start.SessionID, "user", text); err != nil {
				return fmt.Errorf("save user turn: %w", err)
			}
			if err := s.health.SaveConversationTurn(grpcCtx, state.Patient.PatientID, start.SessionID, "assistant", reply); err != nil {
				return fmt.Errorf("save assistant turn: %w", err)
			}
		}
	}
}

// collectUtteranceText buffers PCM from multiple chunks and finalizes when a silence chunk arrives.
// It returns (text, ok, err) where ok==true indicates a full utterance is ready.
func (s *AgentWorkerService) collectUtteranceText(
	ctx context.Context,
	state *models.SessionState,
	wavChunk []byte,
	language string,
) (string, bool, error) {
	if state == nil {
		return "", false, errors.New("state is required")
	}

	// Lazy-init buffers using function statics stored in state via closures is not an option;
	// keep them on the stack via package-level vars? Not acceptable. We'll use state.LastUserText
	// only for transcript de-dupe, and keep utterance audio buffers as function-level statics
	// in a map keyed by session ID.
	//
	// For simplicity and correctness (no global state), we keep buffers in state using hidden fields
	// by encoding into LastAssistantText? Not acceptable either.
	//
	// Instead, use a small in-memory buffer map on the service instance.
	//
	// NOTE: AgentWorkerService is per-process singleton in this repo, so this is safe.
	//
	// This method is implemented as a wrapper around per-room buffers in the room gateway,
	// but since core shouldn't depend on LiveKit types, we maintain a minimal per-session buffer here.
	return s.utteranceCollector().collect(ctx, state, wavChunk, language, s.stt)
}

type utteranceCollector struct {
	mu        sync.Mutex
	bySession map[string]*utteranceBuffer
}

type utteranceBuffer struct {
	pcm        []int16
	startedAt  time.Time
	lastSpeech time.Time
}

func (s *AgentWorkerService) utteranceCollector() *utteranceCollector {
	// Store on the service instance via a private package-level singleton to avoid changing struct fields.
	// This is intentionally process-local and cleared when a session ends.
	return defaultUtteranceCollector
}

var defaultUtteranceCollector = func() *utteranceCollector {
	return &utteranceCollector{bySession: make(map[string]*utteranceBuffer)}
}()

func (c *utteranceCollector) delete(sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	c.mu.Lock()
	delete(c.bySession, sessionID)
	c.mu.Unlock()
}

func (c *utteranceCollector) collect(
	ctx context.Context,
	state *models.SessionState,
	wavChunk []byte,
	language string,
	stt outbound.Transcriber,
) (string, bool, error) {
	now := time.Now()
	isSilence := isMostlySilenceWAV(wavChunk)

	c.mu.Lock()
	buf := c.bySession[state.SessionID]
	if buf == nil {
		buf = &utteranceBuffer{}
		c.bySession[state.SessionID] = buf
	}
	c.mu.Unlock()

	if !isSilence {
		pcm, ok := wavToPCM16(wavChunk)
		if ok {
			if buf.startedAt.IsZero() {
				buf.startedAt = now
			}
			buf.lastSpeech = now
			buf.pcm = append(buf.pcm, pcm...)
			// Hard cap to avoid unbounded buffering if silence detection fails.
			if buf.startedAt.Add(maxUtteranceDuration()).Before(now) {
				return c.flush(ctx, state, language, stt)
			}
		}
		return "", false, nil
	}

	// Silence chunk: if we have buffered speech, finalize.
	if len(buf.pcm) == 0 {
		return "", false, nil
	}
	// Require that we actually saw speech recently to avoid flushing on initial silence.
	// We want to avoid cutting the caller off mid-utterance; require ~1s of silence
	// (with 500ms chunks, that's typically 2 consecutive silence chunks).
	if !buf.lastSpeech.IsZero() && now.Sub(buf.lastSpeech) < 900*time.Millisecond {
		return "", false, nil
	}

	return c.flush(ctx, state, language, stt)
}

func maxUtteranceDuration() time.Duration {
	return 12 * time.Second
}

func (c *utteranceCollector) flush(ctx context.Context, state *models.SessionState, language string, stt outbound.Transcriber) (string, bool, error) {
	c.mu.Lock()
	buf := c.bySession[state.SessionID]
	if buf == nil || len(buf.pcm) == 0 {
		c.mu.Unlock()
		return "", false, nil
	}
	pcm := make([]int16, len(buf.pcm))
	copy(pcm, buf.pcm)
	buf.pcm = nil
	buf.startedAt = time.Time{}
	buf.lastSpeech = time.Time{}
	c.mu.Unlock()

	wav, err := pcm16ToWAVFromInt16(pcm, sttSampleRate, sttChannels)
	if err != nil {
		return "", false, err
	}

	text, err := stt.Transcribe(wav, language)
	if err != nil {
		return "", false, nil
	}
	if strings.TrimSpace(text) == "" {
		return "", false, nil
	}
	if shouldIgnoreTranscript(state, text) {
		return "", false, nil
	}
	return text, true, nil
}

func wavToPCM16(wav []byte) ([]int16, bool) {
	const minWAVHeader = 44
	if len(wav) < minWAVHeader {
		return nil, false
	}
	dataOff, dataLen := wavDataChunk(wav)
	if dataOff == 0 || dataLen < 2 || dataOff+dataLen > len(wav) {
		return nil, false
	}
	pcm := wav[dataOff : dataOff+dataLen]
	if len(pcm)%2 != 0 {
		return nil, false
	}
	out := make([]int16, 0, len(pcm)/2)
	for i := 0; i+1 < len(pcm); i += 2 {
		out = append(out, int16(binary.LittleEndian.Uint16(pcm[i:i+2])))
	}
	return out, true
}

func pcm16ToWAVFromInt16(sample []int16, sampleRate, channels int) ([]byte, error) {
	if sampleRate <= 0 || channels <= 0 {
		return nil, errors.New("invalid wav params")
	}
	dataLen := len(sample) * 2
	byteRate := sampleRate * channels * 2
	blockAlign := channels * 2

	buf := bytes.NewBuffer(make([]byte, 0, 44+dataLen))
	buf.WriteString("RIFF")
	if err := binary.Write(buf, binary.LittleEndian, uint32(36+dataLen)); err != nil {
		return nil, err
	}
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	if err := binary.Write(buf, binary.LittleEndian, uint32(16)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(channels)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(byteRate)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(blockAlign)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(16)); err != nil {
		return nil, err
	}
	buf.WriteString("data")
	if err := binary.Write(buf, binary.LittleEndian, uint32(dataLen)); err != nil {
		return nil, err
	}
	for _, s := range sample {
		if err := binary.Write(buf, binary.LittleEndian, s); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func filterToolsByName(tools []models.ToolDef, allowed map[string]bool) []models.ToolDef {
	out := make([]models.ToolDef, 0, len(tools))
	for _, t := range tools {
		if allowed[t.Name] {
			out = append(out, t)
		}
	}
	return out
}

func requiresIdentity(text string) bool {
	t := normalizeTranscript(text)
	if t == "" {
		return false
	}
	// If the caller is asking about their own records/results/meds, we need verification.
	needles := []string{
		"my profile", "my account",
		"my record", "my records", "my results", "test result", "lab result", "lab work",
		"blood pressure record", "blood pressure records", "my blood pressure", "my bp", "my vitals",
		"my medication", "my meds", "my prescription", "refill", "pharmacy",
		"my appointment", "schedule", "reschedule", "cancel appointment",
		"my doctor", "my chart", "patient portal",
	}
	for _, n := range needles {
		if strings.Contains(t, n) {
			return true
		}
	}
	return false
}

func shouldIgnoreTranscript(state *models.SessionState, text string) bool {
	now := time.Now()
	normalized := normalizeTranscript(text)
	if normalized == "" {
		return true
	}

	// De-dupe identical user transcripts that can happen when we read the same buffered audio chunk repeatedly.
	if state.LastUserText != "" && normalized == normalizeTranscript(state.LastUserText) && now.Sub(state.LastUserAt) < 5*time.Second {
		return true
	}

	// Echo suppression: while we're speaking, ignore transcripts that look like our own last reply.
	if !state.SpeakingUntil.IsZero() && now.Before(state.SpeakingUntil) && state.LastAssistantText != "" {
		asst := normalizeTranscript(state.LastAssistantText)
		// Fast path: substring match (common for echoed greetings).
		if asst != "" && (strings.Contains(asst, normalized) || strings.Contains(normalized, asst)) {
			return true
		}
		// Slightly slower path: word overlap heuristic.
		if tokenOverlapRatio(asst, normalized) >= 0.7 {
			return true
		}
	}

	state.LastUserText = text
	state.LastUserAt = now
	return false
}

func markAssistantSpeaking(state *models.SessionState, reply string, pcm []byte) {
	state.LastAssistantText = reply

	// ElevenLabs adapter returns raw PCM bytes when output_format starts with "pcm_".
	// We currently publish to LiveKit with sampleRate=24000, channels=1, PCM16 => 48000 bytes/sec.
	const bytesPerSecond = 24000 * 1 * 2
	d := 500 * time.Millisecond
	if bytesPerSecond > 0 && len(pcm) > 0 {
		sec := float64(len(pcm)) / float64(bytesPerSecond)
		if sec > 0 {
			d = time.Duration(sec*float64(time.Second)) + 500*time.Millisecond
		}
	}
	state.SpeakingUntil = time.Now().Add(d)
}

func normalizeTranscript(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// Keep letters/digits/spaces. Drop punctuation to make echo matching robust.
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastSpace = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSpace = false
		case r == ' ' || r == '\n' || r == '\t' || r == '\r':
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func tokenOverlapRatio(a, b string) float64 {
	aw := strings.Fields(a)
	bw := strings.Fields(b)
	if len(aw) == 0 || len(bw) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(aw))
	for _, w := range aw {
		set[w] = struct{}{}
	}
	overlap := 0
	for _, w := range bw {
		if _, ok := set[w]; ok {
			overlap++
		}
	}
	den := float64(len(aw))
	if len(bw) > len(aw) {
		den = float64(len(bw))
	}
	if den == 0 {
		return 0
	}
	return float64(overlap) / den
}

// isMostlySilenceWAV performs a tiny energy gate on 16-bit PCM WAV payloads.
// This prevents STT from being invoked on pure silence/background noise, which can
// otherwise produce repeated short transcripts and trigger the greeting loop.
func isMostlySilenceWAV(wav []byte) bool {
	const (
		minWAVHeader = 44
		// Tuned conservatively: we only skip when the average absolute amplitude is very low.
		avgAbsThreshold = 250.0
		minSamples      = 24000 / 4 // ~250ms at 24kHz mono
	)
	if len(wav) < minWAVHeader {
		return false
	}
	dataOff, dataLen := wavDataChunk(wav)
	if dataOff == 0 || dataLen < 2 {
		return false
	}
	if dataOff+dataLen > len(wav) {
		return false
	}

	pcm := wav[dataOff : dataOff+dataLen]
	n := len(pcm) / 2
	if n < minSamples {
		return false
	}

	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
		sum += math.Abs(float64(s))
	}
	avg := sum / float64(n)
	return avg < avgAbsThreshold
}

func wavDataChunk(wav []byte) (off int, n int) {
	// Minimal RIFF/WAVE scan: find the first "data" chunk.
	// RIFF header: 12 bytes. Chunks follow: 4-byte id, 4-byte size, then payload.
	if len(wav) < 12 {
		return 0, 0
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0, 0
	}
	i := 12
	for i+8 <= len(wav) {
		id := string(wav[i : i+4])
		sz := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
		i += 8
		if i+sz > len(wav) {
			return 0, 0
		}
		if id == "data" {
			return i, sz
		}
		// chunks are word-aligned
		i += sz
		if sz%2 == 1 {
			i++
		}
	}
	return 0, 0
}

func (s *AgentWorkerService) runLLMLoop(
	ctx context.Context,
	systemPrompt string,
	tools []models.ToolDef,
	state *models.SessionState,
	messages []models.Message,
) (string, []models.Message, error) {
	currentPrompt := systemPrompt
	currentTools := tools
	for {
		resp, err := s.llm.Chat(ctx, currentPrompt, messages, currentTools)
		if err != nil {
			return "", messages, err
		}

		if resp.Text != "" {
			messages = append(messages, models.Message{
				Role:    "assistant",
				Content: resp.Text,
			})
			return resp.Text, messages, nil
		}

		if resp.ToolUse == nil {
			return "", messages, errors.New("llm returned neither text nor tool use")
		}

		callArgs := make(map[string]any, len(resp.ToolUse.Args)+1)
		for k, v := range resp.ToolUse.Args {
			callArgs[k] = v
		}
		if state.Token != "" {
			callArgs["_auth"] = state.Token
		}
		if resp.ToolUse.Name == "get_location" {
			if _, ok := callArgs["sessionID"]; !ok {
				callArgs["sessionID"] = state.SessionID
			}
		}

		llmArgs := make(map[string]any, len(callArgs))
		for k, v := range callArgs {
			if k == "_auth" {
				continue
			}
			llmArgs[k] = v
		}

		toolCall := &models.ToolCall{
			ID:   resp.ToolUse.ID,
			Name: resp.ToolUse.Name,
			Args: llmArgs,
		}

		result := ""
		if isLocalTool(resp.ToolUse.Name) {
			var upgraded bool
			result, upgraded, err = s.callLocalTool(ctx, resp.ToolUse.Name, callArgs, state)
			if upgraded {
				currentPrompt, currentTools = s.llm.BuildAuthenticatedSystemPrompt(state.Patient.PatientID, state.Language, state.Context), s.llm.BuildAuthenticatedToolDefinitions()
			}
		} else {
			result, err = s.mcp.CallTool(ctx, resp.ToolUse.Name, callArgs)
		}
		if err != nil {
			result = "tool error: " + err.Error()
		}

		messages = append(messages,
			models.Message{
				Role:     "assistant",
				ToolCall: toolCall,
			},
			models.Message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    result,
			},
		)
	}
}

func (s *AgentWorkerService) discoverPatientCandidate(ctx context.Context, state *models.SessionState) error {
	if state.Patient != nil || state.PatientCandidate != nil {
		return nil
	}
	if state.PhoneLookupAttempted || strings.TrimSpace(state.CallerPhone) == "" {
		return nil
	}

	state.PhoneLookupAttempted = true
	candidate, err := s.patientIdentity.LookupByPhone(ctx, state.CallerPhone)
	if err != nil {
		return err
	}
	if candidate == nil || strings.TrimSpace(candidate.PatientID) == "" {
		log.Printf("agent-worker patient candidate not found room=%s session=%s caller=%s method=phone", state.RoomName, state.SessionID, state.CallerPhone)
		return nil
	}

	state.PatientCandidate = candidate
	log.Printf("agent-worker patient candidate found room=%s session=%s patient=%s method=phone", state.RoomName, state.SessionID, candidate.PatientID)
	return nil
}

func (s *AgentWorkerService) upgradeToPatientSession(
	ctx context.Context,
	state *models.SessionState,
) (string, []models.ToolDef, error) {
	if state.Patient == nil || strings.TrimSpace(state.Patient.PatientID) == "" {
		return "", nil, errors.New("identified patient is required")
	}

	token, err := s.tokenIssuer.MintSessionToken(
		state.Patient.PatientID,
		state.SessionID,
		[]string{"records:read", "location:read", "emergency:write"},
	)
	if err != nil {
		return "", nil, fmt.Errorf("mint session token: %w", err)
	}

	grpcCtx := grpcclient.WithForwardedToken(ctx, token)
	contextText, err := s.health.LoadRecentContext(grpcCtx, state.Patient.PatientID, 10)
	if err != nil {
		return "", nil, fmt.Errorf("load recent context: %w", err)
	}

	state.Token = token
	state.Context = contextText

	log.Printf("agent-worker patient session upgraded room=%s session=%s patient=%s", state.RoomName, state.SessionID, state.Patient.PatientID)

	return s.llm.BuildAuthenticatedSystemPrompt(state.Patient.PatientID, state.Language, contextText), s.llm.BuildAuthenticatedToolDefinitions(), nil
}

func isLocalTool(name string) bool {
	switch name {
	case "request_verification_code", "start_registration", "verify_phone_otp":
		return true
	default:
		return false
	}
}

func (s *AgentWorkerService) callLocalTool(ctx context.Context, name string, args map[string]any, state *models.SessionState) (string, bool, error) {
	switch name {
	case "request_verification_code":
		if strings.TrimSpace(state.CallerPhone) == "" {
			err := errors.New("caller phone number is unavailable")
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		if state.Patient != nil {
			return "The caller is already verified.", false, nil
		}
		if state.RegistrationToken != "" {
			state.VerificationMode = "registration"
			return "A registration verification code was already sent. Ask the caller for the code.", false, nil
		}
		if err := s.discoverPatientCandidate(ctx, state); err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		if state.PatientCandidate == nil {
			return "No existing patient was found for this phone number. If the caller wants to register as a new patient, collect their full name and date of birth, then use start_registration.", false, nil
		}
		if err := s.patientIdentity.StartExistingPhoneVerification(ctx, state.CallerPhone); err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		state.VerificationMode = "existing"
		return "A verification code was sent to the caller's phone. Ask the caller for the code.", false, nil
	case "start_registration":
		if strings.TrimSpace(state.CallerPhone) == "" {
			err := errors.New("caller phone number is unavailable")
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		req, err := registrationRequestFromArgs(args, state.CallerPhone)
		if err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		token, err := s.patientIdentity.StartRegistration(ctx, req)
		if err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		state.RegistrationToken = token
		state.VerificationMode = "registration"
		log.Printf("agent-worker registration started room=%s session=%s caller=%s token_present=%t", state.RoomName, state.SessionID, state.CallerPhone, token != "")
		return "Registration started and a verification code was sent to the caller's phone. Ask the caller for the code.", false, nil
	case "verify_phone_otp":
		otp, err := requiredStringArg(args, "otp")
		if err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		var patient *models.IdentifiedPatient
		switch state.VerificationMode {
		case "existing":
			patient, err = s.patientIdentity.VerifyExistingPhoneOTP(ctx, state.CallerPhone, otp)
		case "registration":
			patient, err = s.patientIdentity.VerifyRegistrationOTPAndCreatePatient(ctx, state.CallerPhone, otp, state.RegistrationToken)
		default:
			err := errors.New("no phone verification is pending")
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		if err != nil {
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s mode=%s: %v", name, state.RoomName, state.SessionID, state.CallerPhone, state.VerificationMode, err)
			return "", false, err
		}
		state.Patient = patient
		state.PatientCandidate = nil
		state.VerificationMode = ""
		state.RegistrationToken = ""
		if _, _, err := s.upgradeToPatientSession(ctx, state); err != nil {
			state.Patient = nil
			state.Token = ""
			state.Context = ""
			log.Printf("agent-worker verification tool failed tool=%s room=%s session=%s caller=%s: upgrade failed: %v", name, state.RoomName, state.SessionID, state.CallerPhone, err)
			return "", false, err
		}
		log.Printf("agent-worker identity verified room=%s session=%s caller=%s patient=%s", state.RoomName, state.SessionID, state.CallerPhone, state.Patient.PatientID)
		return "Identity verified successfully. Patient context is now available.", true, nil
	default:
		return "", false, fmt.Errorf("unsupported local tool %q", name)
	}
}

func requiredStringArg(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	str, ok := value.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return strings.TrimSpace(str), nil
}

func registrationRequestFromArgs(args map[string]any, callerPhone string) (models.RegistrationRequest, error) {
	fullName, err := requiredStringArg(args, "fullName")
	if err != nil {
		return models.RegistrationRequest{}, err
	}
	dateOfBirthStr, err := requiredStringArg(args, "dateOfBirth")
	if err != nil {
		return models.RegistrationRequest{}, err
	}
	dateOfBirth, err := parseDateOfBirth(dateOfBirthStr)
	if err != nil {
		return models.RegistrationRequest{}, err
	}

	email := ""
	if raw, ok := args["email"]; ok {
		if str, ok := raw.(string); ok {
			email = strings.TrimSpace(str)
		}
	}

	return models.RegistrationRequest{
		PhoneNumber: callerPhone,
		Email:       email,
		FullName:    fullName,
		DateOfBirth: dateOfBirth,
	}, nil
}

func parseDateOfBirth(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("dateOfBirth is required")
	}

	// Normalize common separators to '-' to reduce layout permutations.
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.Join(strings.Fields(s), " ")

	// Preferred: ISO date.
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Handle digit-only or partially-delimited formats often produced by STT:
	//   "0102-2005" (MMDD-YYYY) -> 01-02-2005
	//   "01022005"  (MMDDYYYY)  -> 01-02-2005
	//   "20050102"  (YYYYMMDD)  -> 2005-01-02
	if t, ok := parseDOBDigitsOnly(s); ok {
		return t, nil
	}

	// Extract numeric components even if STT inserts spaces ("01 - 02 - 2005", "01 02 2005").
	if a, b, c, ok := parseDOBNumericTriplet(s); ok {
		month, day, year := 0, 0, c
		switch {
		case a > 12 && b >= 1 && b <= 12:
			day, month = a, b // DD-MM-YYYY
		case b > 12 && a >= 1 && a <= 12:
			month, day = a, b // MM-DD-YYYY
		default:
			month, day = a, b // ambiguous, assume US MM-DD-YYYY
		}
		if month >= 1 && month <= 12 && day >= 1 && day <= 31 && year >= 1900 && year <= time.Now().Year() {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
		}
	}

	// If the input is numeric with '-' separators, infer order for common US usage.
	// Examples:
	//   01-02-2005 (assume MM-DD-YYYY in US unless unambiguous)
	//   13-02-2005 (DD-MM-YYYY because month can't be 13)
	parts := strings.Split(s, "-")
	if len(parts) == 3 && len(parts[2]) == 4 {
		a, errA := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, errB := strconv.Atoi(strings.TrimSpace(parts[1]))
		c, errC := strconv.Atoi(strings.TrimSpace(parts[2]))
		if errA == nil && errB == nil && errC == nil {
			month, day, year := 0, 0, c
			switch {
			case a > 12 && b >= 1 && b <= 12:
				// DD-MM-YYYY
				day, month = a, b
			case b > 12 && a >= 1 && a <= 12:
				// MM-DD-YYYY
				month, day = a, b
			default:
				// Ambiguous (both <= 12). Assume MM-DD-YYYY for US callers.
				month, day = a, b
			}

			if month >= 1 && month <= 12 && day >= 1 && day <= 31 && year >= 1900 && year <= time.Now().Year() {
				// Construct a date in UTC.
				return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
			}
		}
	}

	// Try a few text layouts for callers who say month names.
	for _, layout := range []string{
		"January 2 2006",
		"Jan 2 2006",
		"2 January 2006",
		"2 Jan 2006",
		"January 2, 2006",
		"Jan 2, 2006",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			// Strip time component.
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("dateOfBirth must be a valid date (prefer YYYY-MM-DD; also accepted: MM-DD-YYYY, MM/DD/YYYY)")
}

var dobTripletRe = regexp.MustCompile(`(?i)(\d{1,2})\D+(\d{1,2})\D+(\d{4})`)

func parseDOBNumericTriplet(s string) (a, b, c int, ok bool) {
	m := dobTripletRe.FindStringSubmatch(s)
	if len(m) != 4 {
		return 0, 0, 0, false
	}
	aa, errA := strconv.Atoi(m[1])
	bb, errB := strconv.Atoi(m[2])
	cc, errC := strconv.Atoi(m[3])
	if errA != nil || errB != nil || errC != nil {
		return 0, 0, 0, false
	}
	return aa, bb, cc, true
}

func parseDOBDigitsOnly(s string) (time.Time, bool) {
	digits := digitsOnly(s)
	if len(digits) != 8 {
		return time.Time{}, false
	}

	nowYear := time.Now().Year()

	// YYYYMMDD
	if y, err := strconv.Atoi(digits[0:4]); err == nil && y >= 1900 && y <= nowYear {
		mm, errM := strconv.Atoi(digits[4:6])
		dd, errD := strconv.Atoi(digits[6:8])
		if errM == nil && errD == nil && mm >= 1 && mm <= 12 && dd >= 1 && dd <= 31 {
			return time.Date(y, time.Month(mm), dd, 0, 0, 0, 0, time.UTC), true
		}
	}

	// MMDDYYYY or DDMMYYYY
	y, errY := strconv.Atoi(digits[4:8])
	if errY != nil || y < 1900 || y > nowYear {
		return time.Time{}, false
	}
	a, errA := strconv.Atoi(digits[0:2])
	b, errB := strconv.Atoi(digits[2:4])
	if errA != nil || errB != nil {
		return time.Time{}, false
	}
	month, day := 0, 0
	switch {
	case a > 12 && b >= 1 && b <= 12:
		day, month = a, b // DDMMYYYY
	case b > 12 && a >= 1 && a <= 12:
		month, day = a, b // MMDDYYYY
	default:
		month, day = a, b // ambiguous, assume US MMDDYYYY
	}
	if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
		return time.Date(y, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
	}
	return time.Time{}, false
}

func digitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
