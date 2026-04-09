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
	provider      outbound.TranslationProvider
	maxTextLength int
}

func NewTranslationService(provider outbound.TranslationProvider, maxTextLength int) *TranslationService {
	return &TranslationService{
		provider:      provider,
		maxTextLength: maxTextLength,
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
			TranslatedText: req.Text,
			DetectedLang:   strings.ToLower(req.SourceLang),
			CharacterCount: utf8.RuneCountInString(req.Text),
		}, nil
	}

	return s.provider.Translate(ctx, req)
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
