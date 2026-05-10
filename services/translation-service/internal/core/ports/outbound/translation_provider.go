package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
)

// TranslationProvider is the outbound port for provider-backed translation.
// The first implementation will use DeepL, but tests and future providers
// can satisfy the same contract without changing the core service.
type TranslationProvider interface {
	Translate(ctx context.Context, req models.TranslationRequest) (*models.TranslationResult, error)
}
