package services

import (
	"context"
	"strings"
	"unicode/utf8"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/ports/outbound"
)

type TranslationService struct {
	provider            outbound.TranslationProvider
	maxTextLength       int
	confidenceThreshold float64
}

const LowConfidenceAdvisory = "Translation confidence is low. Human interpreter recommended."

func NewTranslationService(provider outbound.TranslationProvider, maxTextLength int, confidenceThreshold float64) *TranslationService {
	return &TranslationService{
		provider:            provider,
		maxTextLength:       maxTextLength,
		confidenceThreshold: confidenceThreshold,
	}
}

func (s *TranslationService) Translate(ctx context.Context, req models.TranslationRequest) (*models.TranslationResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, domainerrors.ErrEmptyText
	}
	if utf8.RuneCountInString(req.Text) > s.maxTextLength {
		return nil, domainerrors.ErrTextTooLong
	}

	targetLang, err := normalizeLanguageCode(req.TargetLang)
	if err != nil {
		return nil, err
	}
	req.TargetLang = targetLang

	if strings.TrimSpace(req.SourceLang) != "" {
		sourceLang, err := normalizeLanguageCode(req.SourceLang)
		if err != nil {
			return nil, err
		}
		req.SourceLang = sourceLang
	}

	if req.SourceLang != "" && req.SourceLang == req.TargetLang {
		return &models.TranslationResult{
			TranslatedText:               req.Text,
			DetectedLang:                 strings.ToLower(req.SourceLang),
			SourceLang:                   strings.ToLower(req.SourceLang),
			TargetLang:                   strings.ToLower(req.TargetLang),
			CharacterCount:               utf8.RuneCountInString(req.Text),
			ConfidenceScore:              1.0,
			TranslationProvider:          "identity",
			MedicalTermPreservationCheck: true,
		}, nil
	}

	result, err := s.provider.Translate(ctx, req)
	if err != nil {
		return nil, err
	}
	if result.SourceLang == "" {
		result.SourceLang = strings.ToLower(req.SourceLang)
	}
	if result.TargetLang == "" {
		result.TargetLang = strings.ToLower(req.TargetLang)
	}
	if result.ConfidenceScore <= 0 {
		result.ConfidenceScore = 0.5
	}
	if result.ConfidenceScore < s.confidenceThreshold {
		result.AdvisoryMessage = LowConfidenceAdvisory
	}
	return result, nil
}

func normalizeLanguageCode(code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", domainerrors.ErrInvalidLanguageCode
	}

	normalized, ok := models.SupportedLanguageCodes[strings.ToLower(strings.TrimSpace(code))]
	if !ok {
		return "", domainerrors.ErrUnsupportedLanguage
	}

	return normalized, nil
}
