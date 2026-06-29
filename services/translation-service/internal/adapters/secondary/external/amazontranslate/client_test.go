package amazontranslate

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
	"github.com/aws/aws-sdk-go-v2/service/translate"
)

type fakeAPI struct {
	out *translate.TranslateTextOutput
	err error
}

func (f fakeAPI) TranslateText(_ context.Context, _ *translate.TranslateTextInput, _ ...func(*translate.Options)) (*translate.TranslateTextOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func TestTranslateUsesAutoDetect(t *testing.T) {
	client := NewWithAPI(fakeAPI{
		out: &translate.TranslateTextOutput{
			TranslatedText:     stringPtr("hola"),
			SourceLanguageCode: stringPtr("en"),
		},
	})
	got, err := client.Translate(context.Background(), models.TranslationRequest{
		Text:       "hello",
		TargetLang: "es",
	})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got.TranslatedText != "hola" {
		t.Fatalf("TranslatedText = %q, want hola", got.TranslatedText)
	}
	if got.DetectedLang != "en" {
		t.Fatalf("DetectedLang = %q, want en", got.DetectedLang)
	}
	if got.TranslationProvider != "amazon_translate" {
		t.Fatalf("TranslationProvider = %q, want amazon_translate", got.TranslationProvider)
	}
	if got.ConfidenceScore <= 0 {
		t.Fatalf("ConfidenceScore = %v, want > 0", got.ConfidenceScore)
	}
}

func TestTranslateMapsProviderErrors(t *testing.T) {
	client := NewWithAPI(fakeAPI{err: errors.New("boom")})
	_, err := client.Translate(context.Background(), models.TranslationRequest{
		Text:       "hello",
		TargetLang: "es",
	})
	if !errors.Is(err, domainerrors.ErrProviderUnavailable) {
		t.Fatalf("Translate() error = %v, want provider unavailable", err)
	}
}

func stringPtr(v string) *string {
	return &v
}
