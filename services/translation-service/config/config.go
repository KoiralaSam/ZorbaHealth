package config

import (
	"fmt"
	"time"

	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
)

type Config struct {
	TranslationServiceGRPCAddr string
	TranslationProvider        string
	TranslationModelBaseURL    string
	TranslationModelName       string
	ModelTimeout               time.Duration
	ModelMaxTokens             int
	InternalServiceSecret      string
	MaxTextLength              int
	RateLimitPerMinute         int
	ConfidenceThreshold        float64
	EnableGRPCReflection       bool
	AWSRegion                  string
}

func Load() (*Config, error) {
	cfg := &Config{
		TranslationServiceGRPCAddr: sharedenv.GetString("TRANSLATION_SERVICE_GRPC_ADDR", ":50057"),
		TranslationProvider:        sharedenv.GetString("TRANSLATION_PROVIDER", "llamacpp"),
		TranslationModelBaseURL:    sharedenv.GetString("TRANSLATION_MODEL_BASE_URL", "http://translation-model:8080"),
		TranslationModelName:       sharedenv.GetString("TRANSLATION_MODEL_NAME", ""),
		ModelTimeout:               time.Duration(sharedenv.GetInt("TRANSLATION_MODEL_TIMEOUT_SECONDS", 30)) * time.Second,
		ModelMaxTokens:             sharedenv.GetInt("TRANSLATION_MODEL_MAX_TOKENS", 128),
		InternalServiceSecret:      sharedenv.GetString("INTERNAL_SERVICE_SECRET", ""),
		MaxTextLength:              sharedenv.GetInt("TRANSLATION_SERVICE_MAX_TEXT_LENGTH", 10000),
		RateLimitPerMinute:         sharedenv.GetInt("TRANSLATION_SERVICE_RATE_LIMIT_PER_MINUTE", 60),
		ConfidenceThreshold:        sharedenv.GetFloat("TRANSLATION_CONFIDENCE_THRESHOLD", 0.75),
		EnableGRPCReflection:       sharedenv.GetBool("TRANSLATION_SERVICE_ENABLE_GRPC_REFLECTION", false),
		AWSRegion:                  sharedenv.GetString("AWS_REGION", "us-east-1"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.TranslationServiceGRPCAddr == "" {
		return fmt.Errorf("TRANSLATION_SERVICE_GRPC_ADDR is required")
	}
	if c.InternalServiceSecret == "" {
		return fmt.Errorf("INTERNAL_SERVICE_SECRET is required")
	}
	if len(c.InternalServiceSecret) < 32 {
		return fmt.Errorf("INTERNAL_SERVICE_SECRET must be at least 32 bytes")
	}
	if c.ModelTimeout <= 0 {
		return fmt.Errorf("TRANSLATION_MODEL_TIMEOUT_SECONDS must be greater than zero")
	}
	if c.ModelMaxTokens <= 0 {
		return fmt.Errorf("TRANSLATION_MODEL_MAX_TOKENS must be greater than zero")
	}
	if c.MaxTextLength <= 0 {
		return fmt.Errorf("TRANSLATION_SERVICE_MAX_TEXT_LENGTH must be greater than zero")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("TRANSLATION_SERVICE_RATE_LIMIT_PER_MINUTE must be greater than zero")
	}
	if c.ConfidenceThreshold <= 0 || c.ConfidenceThreshold > 1 {
		return fmt.Errorf("TRANSLATION_CONFIDENCE_THRESHOLD must be between 0 and 1")
	}
	switch c.TranslationProvider {
	case "", "llamacpp":
		if c.TranslationModelBaseURL == "" {
			return fmt.Errorf("TRANSLATION_MODEL_BASE_URL is required")
		}
	case "amazon_translate":
		if c.AWSRegion == "" {
			return fmt.Errorf("AWS_REGION is required for amazon_translate")
		}
	default:
		return fmt.Errorf("unsupported TRANSLATION_PROVIDER %q", c.TranslationProvider)
	}

	return nil
}
