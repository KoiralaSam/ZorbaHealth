package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
)

// TranslationService is the primary inbound port for translation requests.
// Primary adapters such as gRPC handlers depend on this interface.
type TranslationService interface {
	Translate(ctx context.Context, req models.TranslationRequest) (*models.TranslationResult, error)
}
