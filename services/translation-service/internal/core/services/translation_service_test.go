package services

import (
	"context"
	"testing"

	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
)

type stubProvider struct {
	result *models.TranslationResult
	err    error
}

func (s stubProvider) Translate(context.Context, models.TranslationRequest) (*models.TranslationResult, error) {
	return s.result, s.err
}

func TestTranslateAddsLowConfidenceAdvisory(t *testing.T) {
	svc := NewTranslationService(stubProvider{
		result: &models.TranslationResult{
			TranslatedText:      "texto",
			DetectedLang:        "en",
			ConfidenceScore:     0.5,
			TranslationProvider: "llamacpp",
		},
	}, 1000, 0.75)

	got, err := svc.Translate(context.Background(), models.TranslationRequest{
		Text:       "text",
		TargetLang: "es",
		SourceLang: "en",
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.AdvisoryMessage != LowConfidenceAdvisory {
		t.Fatalf("AdvisoryMessage = %q, want %q", got.AdvisoryMessage, LowConfidenceAdvisory)
	}
}
