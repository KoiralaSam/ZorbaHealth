package outbound

import "context"

type Transcriber interface {
	Transcribe(audio []byte, language string) (string, error)
}

type Synthesizer interface {
	Speak(text string, language string) ([]byte, error)
}

type RoomAudioGateway interface {
	NextAudioChunk(ctx context.Context, roomName string) ([]byte, error)
	PublishAudio(ctx context.Context, roomName string, audio []byte) error
}
