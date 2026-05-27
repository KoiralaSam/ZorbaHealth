package models

import "github.com/google/uuid"

// ScoredChunk represents a record chunk returned from a vector similarity search.
type ScoredChunk struct {
	ChunkID          uuid.UUID
	RecordID         uuid.UUID
	FHIRResourceType string
	Text             string
	SourceFile       string
	Score            float32
}

// Turn represents a single conversation turn from voice agent sessions.
type Turn struct {
	Role    string
	Content string
}

// FHIRResourceSummary is a lightweight view used during RAG retrieval.
type FHIRResourceSummary struct {
	ID           uuid.UUID
	ResourceType string
	ResourceID   string
	DisplayText  string
	Status       string
}
