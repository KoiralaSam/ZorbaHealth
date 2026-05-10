package livekit

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/livekit/media-sdk"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	lkmedia "github.com/livekit/server-sdk-go/v2/pkg/media"
	"github.com/pion/webrtc/v4"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
)

const (
	defaultRoomSampleRate    = 24000
	defaultRoomChannels      = 1
	defaultChunkDuration     = 500 * time.Millisecond
	defaultInboundChunkQueue = 8
	connectTimeout          = 10 * time.Second
)

type RoomGateway struct {
	mu                sync.Mutex
	sessions          map[string]*roomSession
	wsURL             string
	apiKey            string
	apiSecret         string
	participantPrefix string
	sampleRate        int
	channels          int
	chunkSamples      int
}

type roomSession struct {
	room         *lksdk.Room
	audioIn      chan []byte
	publishTrack *lkmedia.PCMLocalTrack

	mu        sync.Mutex
	remotePCM *lkmedia.PCMRemoteTrack
	closed    atomic.Bool
	closeOnce sync.Once
	closeFn   func()
}

type pcmChunkWriter struct {
	out          chan<- []byte
	sampleRate   int
	channels     int
	flushSamples int

	mu     sync.Mutex
	buffer media.PCM16Sample
	closed atomic.Bool
}

var _ outbound.RoomAudioGateway = (*RoomGateway)(nil)

func NewRoomGateway() *RoomGateway {
	sampleRate := sharedenv.GetInt("LIVEKIT_AUDIO_SAMPLE_RATE", defaultRoomSampleRate)
	if sampleRate <= 0 {
		sampleRate = defaultRoomSampleRate
	}
	channels := sharedenv.GetInt("LIVEKIT_AUDIO_CHANNELS", defaultRoomChannels)
	if channels <= 0 {
		channels = defaultRoomChannels
	}

	chunkSamples := int(float64(sampleRate*channels) * defaultChunkDuration.Seconds())
	if chunkSamples <= 0 {
		chunkSamples = sampleRate * channels / 2
	}

	return &RoomGateway{
		sessions:          make(map[string]*roomSession),
		wsURL:             strings.TrimSpace(sharedenv.GetString("LIVEKIT_WS_URL", "")),
		apiKey:            strings.TrimSpace(os.Getenv("LIVEKIT_API_KEY")),
		apiSecret:         strings.TrimSpace(os.Getenv("LIVEKIT_API_SECRET")),
		participantPrefix: sharedenv.GetString("LIVEKIT_PARTICIPANT_PREFIX", "agent-worker"),
		sampleRate:        sampleRate,
		channels:          channels,
		chunkSamples:      chunkSamples,
	}
}

func (g *RoomGateway) NextAudioChunk(ctx context.Context, roomName string) ([]byte, error) {
	session, err := g.ensureRoomSession(ctx, roomName)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case chunk, ok := <-session.audioIn:
		if !ok {
			return nil, io.EOF
		}
		log.Printf("livekit room=%s received audio chunk bytes=%d", roomName, len(chunk))
		return chunk, nil
	}
}

func (g *RoomGateway) PublishAudio(ctx context.Context, roomName string, audio []byte) error {
	session, err := g.ensureRoomSession(ctx, roomName)
	if err != nil {
		return err
	}
	if len(audio) == 0 {
		return nil
	}

	sample, err := bytesToPCM16(audio)
	if err != nil {
		return err
	}

	return session.publishTrack.WriteSample(sample)
}

func (g *RoomGateway) ensureRoomSession(ctx context.Context, roomName string) (*roomSession, error) {
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		return nil, errors.New("room name is required")
	}
	if g.wsURL == "" {
		return nil, errors.New("LIVEKIT_WS_URL is not set")
	}
	if g.apiKey == "" || g.apiSecret == "" {
		return nil, errors.New("LIVEKIT_API_KEY and LIVEKIT_API_SECRET are required")
	}

	g.mu.Lock()
	if session, ok := g.sessions[roomName]; ok {
		g.mu.Unlock()
		return session, nil
	}

	session := &roomSession{
		audioIn: make(chan []byte, defaultInboundChunkQueue),
	}
	g.sessions[roomName] = session
	g.mu.Unlock()

	callbacks := lksdk.NewRoomCallback()
	callbacks.OnParticipantConnected = func(rp *lksdk.RemoteParticipant) {
		log.Printf("livekit room=%s remote participant connected identity=%s sid=%s", roomName, rp.Identity(), rp.SID())
	}
	callbacks.ParticipantCallback.OnTrackPublished = func(publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if publication.Kind() != lksdk.TrackKindAudio {
			return
		}
		log.Printf(
			"livekit room=%s audio track published participant=%s track_sid=%s subscribed=%t",
			roomName,
			rp.Identity(),
			publication.SID(),
			publication.IsSubscribed(),
		)
		if !publication.IsSubscribed() {
			if err := publication.SetSubscribed(true); err != nil {
				logger.GetLogger().Errorw("failed to subscribe to livekit remote audio track", err, "room", roomName, "participant", rp.Identity(), "track", publication.SID())
				return
			}
			log.Printf("livekit room=%s requested subscription for participant=%s track_sid=%s", roomName, rp.Identity(), publication.SID())
		}
	}
	callbacks.ParticipantCallback.OnTrackSubscribed = func(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		log.Printf("livekit room=%s audio track subscribed participant=%s track_sid=%s", roomName, rp.Identity(), publication.SID())
		session.mu.Lock()
		defer session.mu.Unlock()
		if session.closed.Load() || session.remotePCM != nil {
			return
		}

		writer := newPCMChunkWriter(session.audioIn, g.sampleRate, g.channels, g.chunkSamples)
		pcmTrack, err := lkmedia.NewPCMRemoteTrack(
			track,
			writer,
			lkmedia.WithTargetSampleRate(g.sampleRate),
			lkmedia.WithTargetChannels(g.channels),
		)
		if err != nil {
			logger.GetLogger().Errorw("failed to create livekit PCM remote track", err, "room", roomName, "participant", rp.Identity())
			return
		}
		session.remotePCM = pcmTrack
		_ = publication
	}
	callbacks.OnDisconnected = func() {
		session.close()
	}
	callbacks.OnDisconnectedWithReason = func(reason lksdk.DisconnectionReason) {
		logger.GetLogger().Infow("livekit room disconnected", "room", roomName, "reason", string(reason))
		session.close()
	}

	log.Printf("livekit attempting room connection room=%s ws_url=%s participant=%s", roomName, g.wsURL, g.participantIdentity(roomName))
	room, err := lksdk.ConnectToRoom(g.wsURL, lksdk.ConnectInfo{
		APIKey:              g.apiKey,
		APISecret:           g.apiSecret,
		RoomName:            roomName,
		ParticipantName:     "Zorba Agent Worker",
		ParticipantIdentity: g.participantIdentity(roomName),
		ParticipantKind:     lksdk.ParticipantAgent,
	}, callbacks, lksdk.WithAutoSubscribe(true), lksdk.WithConnectTimeout(connectTimeout))
	if err != nil {
		g.mu.Lock()
		delete(g.sessions, roomName)
		g.mu.Unlock()
		return nil, fmt.Errorf("connect to livekit room %q: %w", roomName, err)
	}
	log.Printf("livekit room connected room=%s local_participant=%s", roomName, room.LocalParticipant.Identity())

	for _, participant := range room.GetRemoteParticipants() {
		log.Printf("livekit room=%s existing remote participant identity=%s sid=%s", roomName, participant.Identity(), participant.SID())
		for _, publication := range participant.TrackPublications() {
			remotePub, ok := publication.(*lksdk.RemoteTrackPublication)
			if !ok || remotePub.Kind() != lksdk.TrackKindAudio {
				continue
			}
			log.Printf(
				"livekit room=%s existing audio publication participant=%s track_sid=%s subscribed=%t",
				roomName,
				participant.Identity(),
				remotePub.SID(),
				remotePub.IsSubscribed(),
			)
			if !remotePub.IsSubscribed() {
				if err := remotePub.SetSubscribed(true); err != nil {
					logger.GetLogger().Errorw("failed to subscribe to existing livekit audio track", err, "room", roomName, "participant", participant.Identity(), "track", remotePub.SID())
				} else {
					log.Printf("livekit room=%s requested subscription for existing participant=%s track_sid=%s", roomName, participant.Identity(), remotePub.SID())
				}
			}
		}
	}

	publishTrack, err := lkmedia.NewPCMLocalTrack(g.sampleRate, g.channels, logger.GetLogger())
	if err != nil {
		room.Disconnect()
		g.mu.Lock()
		delete(g.sessions, roomName)
		g.mu.Unlock()
		return nil, fmt.Errorf("create livekit publish track: %w", err)
	}
	if _, err := room.LocalParticipant.PublishTrack(publishTrack, &lksdk.TrackPublicationOptions{
		Name: "agent-worker-audio",
	}); err != nil {
		_ = publishTrack.Close()
		room.Disconnect()
		g.mu.Lock()
		delete(g.sessions, roomName)
		g.mu.Unlock()
		return nil, fmt.Errorf("publish livekit audio track: %w", err)
	}

	session.room = room
	session.publishTrack = publishTrack
	session.closeFn = func() {
		g.mu.Lock()
		delete(g.sessions, roomName)
		g.mu.Unlock()

		session.mu.Lock()
		if session.remotePCM != nil {
			session.remotePCM.Close()
			session.remotePCM = nil
		}
		session.mu.Unlock()

		if session.publishTrack != nil {
			session.publishTrack.ClearQueue()
			_ = session.publishTrack.Close()
		}
		if session.room != nil {
			session.room.Disconnect()
		}
		close(session.audioIn)
	}

	return session, nil
}

func (g *RoomGateway) participantIdentity(roomName string) string {
	var b strings.Builder
	b.WriteString(g.participantPrefix)
	b.WriteByte('-')
	for _, r := range roomName {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (s *roomSession) close() {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.closeFn != nil {
			s.closeFn()
		}
	})
}

func newPCMChunkWriter(out chan<- []byte, sampleRate, channels, flushSamples int) *pcmChunkWriter {
	return &pcmChunkWriter{
		out:          out,
		sampleRate:   sampleRate,
		channels:     channels,
		flushSamples: flushSamples,
	}
}

func (w *pcmChunkWriter) WriteSample(sample media.PCM16Sample) error {
	if w.closed.Load() {
		return io.EOF
	}
	if len(sample) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, sample...)
	for len(w.buffer) >= w.flushSamples {
		chunk := make(media.PCM16Sample, w.flushSamples)
		copy(chunk, w.buffer[:w.flushSamples])
		w.buffer = append(media.PCM16Sample(nil), w.buffer[w.flushSamples:]...)

		wav, err := pcm16ToWAV(chunk, w.sampleRate, w.channels)
		if err != nil {
			return err
		}
		w.out <- wav
	}

	return nil
}

func (w *pcmChunkWriter) Close() error {
	if !w.closed.CompareAndSwap(false, true) {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) == 0 {
		return nil
	}

	wav, err := pcm16ToWAV(w.buffer, w.sampleRate, w.channels)
	if err != nil {
		return err
	}
	w.out <- wav
	w.buffer = nil
	return nil
}

func bytesToPCM16(raw []byte) (media.PCM16Sample, error) {
	if len(raw)%2 != 0 {
		return nil, errors.New("audio payload length must be even for PCM16")
	}

	sample := make(media.PCM16Sample, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		sample[i/2] = int16(binary.LittleEndian.Uint16(raw[i : i+2]))
	}
	return sample, nil
}

func pcm16ToWAV(sample media.PCM16Sample, sampleRate, channels int) ([]byte, error) {
	if sampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}
	if channels <= 0 {
		return nil, errors.New("channels must be positive")
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
	for _, v := range sample {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
