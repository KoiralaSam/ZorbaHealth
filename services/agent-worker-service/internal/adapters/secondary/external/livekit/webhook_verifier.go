package livekit

import (
	"errors"
	"net/http"
	"strings"

	livekitauth "github.com/livekit/protocol/auth"
	lklivekit "github.com/livekit/protocol/livekit"
	lkwebhook "github.com/livekit/protocol/webhook"
)

type WebhookVerifier struct {
	provider livekitauth.KeyProvider
}

func NewWebhookVerifier(apiKey, apiSecret string) *WebhookVerifier {
	apiKey = strings.TrimSpace(apiKey)
	apiSecret = strings.TrimSpace(apiSecret)
	if apiKey == "" || apiSecret == "" {
		return &WebhookVerifier{}
	}

	return &WebhookVerifier{
		provider: livekitauth.NewSimpleKeyProvider(apiKey, apiSecret),
	}
}

func (v *WebhookVerifier) ReceiveEvent(r *http.Request) (*lklivekit.WebhookEvent, error) {
	if v.provider == nil {
		return nil, errors.New("livekit webhook verifier is not configured")
	}

	return lkwebhook.ReceiveWebhookEvent(r, v.provider)
}
