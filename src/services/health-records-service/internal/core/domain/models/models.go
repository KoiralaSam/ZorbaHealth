package models

// ScoredChunk represents a record chunk returned from a vector similarity search.
type ScoredChunk struct {
	Text       string
	SourceFile string
	Score      float32
}

// Turn represents a single conversation turn (user or assistant) from agent-worker sessions.
type Turn struct {
	Role    string
	Content string
}
