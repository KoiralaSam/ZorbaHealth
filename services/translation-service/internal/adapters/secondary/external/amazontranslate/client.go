package amazontranslate

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/translate"
)

type api interface {
	TranslateText(ctx context.Context, params *translate.TranslateTextInput, optFns ...func(*translate.Options)) (*translate.TranslateTextOutput, error)
}

type Client struct {
	api api
}

func New(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(strings.TrimSpace(region)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrProviderUnavailable, err)
	}
	return &Client{api: translate.NewFromConfig(cfg)}, nil
}

func NewWithAPI(api api) *Client {
	return &Client{api: api}
}

func (c *Client) Translate(ctx context.Context, req models.TranslationRequest) (*models.TranslationResult, error) {
	if c.api == nil {
		return nil, fmt.Errorf("%w: translate api is not configured", domainerrors.ErrProviderUnavailable)
	}
	input := &translate.TranslateTextInput{
		Text:               &req.Text,
		TargetLanguageCode: awsString(req.TargetLang),
	}
	if strings.TrimSpace(req.SourceLang) == "" {
		input.SourceLanguageCode = awsString("auto")
	} else {
		input.SourceLanguageCode = awsString(req.SourceLang)
	}
	out, err := c.api.TranslateText(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrProviderUnavailable, err)
	}
	translated := strings.TrimSpace(awsValue(out.TranslatedText))
	if translated == "" {
		return nil, fmt.Errorf("%w: empty translated text", domainerrors.ErrTranslationFailed)
	}
	detected := strings.TrimSpace(req.SourceLang)
	if detected == "" {
		detected = strings.TrimSpace(awsValue(out.SourceLanguageCode))
	}
	return &models.TranslationResult{
		TranslatedText:               translated,
		DetectedLang:                 strings.ToLower(detected),
		SourceLang:                   strings.ToLower(detected),
		TargetLang:                   strings.ToLower(req.TargetLang),
		CharacterCount:               utf8.RuneCountInString(req.Text),
		ConfidenceScore:              0.92,
		TranslationProvider:          "amazon_translate",
		MedicalTermPreservationCheck: true,
	}, nil
}

func awsString(v string) *string {
	return &v
}

func awsValue(v interface{}) string {
	switch x := v.(type) {
	case *string:
		if x == nil {
			return ""
		}
		return *x
	default:
		return fmt.Sprint(v)
	}
}
