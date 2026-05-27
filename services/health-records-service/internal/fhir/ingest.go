package fhir

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/rag/chunker"
)

// Ingestor stores validated FHIR resources and indexes searchable text for RAG.
type Ingestor struct {
	store          outbound.Store
	embedder       outbound.Embedder
	embeddingModel string
	chunker        chunker.Chunker
}

func NewIngestor(store outbound.Store, embedder outbound.Embedder, embeddingModel string) *Ingestor {
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	return &Ingestor{
		store:          store,
		embedder:       embedder,
		embeddingModel: embeddingModel,
		chunker:        chunker.NewWordChunker(500, 50),
	}
}

type IngestResult struct {
	ResourcesStored int32
	ChunksStored    int32
}

func (i *Ingestor) IngestBundle(ctx context.Context, internalPatientID uuid.UUID, bundleJSON, sourceSystem string) (IngestResult, error) {
	resources, err := ValidateBundleJSON(bundleJSON)
	if err != nil {
		return IngestResult{}, err
	}

	var result IngestResult
	for _, raw := range resources {
		norm, err := NormalizeResource(raw)
		if err != nil {
			return result, err
		}

		if norm.ResourceType == "Patient" && norm.FHIRPatientID != "" {
			if err := i.store.UpsertFHIRPatientMap(ctx, norm.FHIRPatientID, sourceSystem, internalPatientID); err != nil {
				return result, err
			}
		}

		recordID, err := i.store.UpsertResourceNormalized(
			ctx,
			internalPatientID,
			norm.ResourceType,
			norm.ResourceID,
			sourceSystem,
			raw,
			norm.DisplayText,
			norm.ClinicalStatus,
			norm.EffectiveDate,
		)
		if err != nil {
			return result, fmt.Errorf("store %s/%s: %w", norm.ResourceType, norm.ResourceID, err)
		}
		result.ResourcesStored++

		chunks := i.chunker.Chunk(norm.SearchableText)
		for idx, text := range chunks {
			embedding, err := i.embedder.Embed(ctx, text)
			if err != nil {
				return result, err
			}
			hash := chunker.Hash(text)
			sourceFile := fmt.Sprintf("%s:%s", norm.ResourceType, sourceSystem)
			if err := i.store.CreateRecordChunkDetailed(ctx, outbound.RecordChunkInput{
				PatientID:        internalPatientID,
				RecordID:         recordID,
				FHIRResourceType: norm.ResourceType,
				SourceSystem:     sourceSystem,
				SourceFile:       sourceFile,
				ChunkIndex:       int32(idx),
				ChunkText:        text,
				ChunkHash:        hash,
				AccessLevel:      "patient",
				EmbeddingModel:   i.embeddingModel,
				Embedding:        embedding,
			}); err != nil {
				return result, err
			}
			result.ChunksStored++
		}
	}
	return result, nil
}

func (i *Ingestor) IngestResource(ctx context.Context, internalPatientID uuid.UUID, resourceJSON json.RawMessage, sourceSystem string) (IngestResult, error) {
	bundle := fmt.Sprintf(`{"resourceType":"Bundle","type":"collection","entry":[{"resource":%s}]}`, string(resourceJSON))
	return i.IngestBundle(ctx, internalPatientID, bundle, sourceSystem)
}
