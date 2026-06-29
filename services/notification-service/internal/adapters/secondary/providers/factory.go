package providers

import (
	"fmt"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/secondary/email"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/secondary/sms/voipms"
	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	sharedproviders "github.com/KoiralaSam/ZorbaHealth/shared/ports/providers"
)

type EmailFactoryConfig struct {
	Provider        string
	APIToken        string
	FromEmail       string
	FromName        string
	SendURL         string
	MirrorRecipient string
}

type SMSFactoryConfig struct {
	Provider string
	BaseURL  string
	Username string
	Password string
	DID      string
}

type EmailSenderProvider interface {
	outbound.EmailSender
	sharedproviders.EmailProvider
}

type SMSSenderProvider interface {
	outbound.SMSSender
	sharedproviders.SMSProvider
}

func NewEmailProvider(cfg EmailFactoryConfig) (EmailSenderProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "mailtrap":
		return email.NewMailtrapSender(cfg.APIToken, cfg.FromEmail, cfg.FromName, cfg.SendURL, cfg.MirrorRecipient), nil
	default:
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrUnsupportedEmailProvider, cfg.Provider)
	}
}

func NewSMSProvider(cfg SMSFactoryConfig) (SMSSenderProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "voipms":
		return voipms.NewSender(cfg.BaseURL, cfg.Username, cfg.Password, cfg.DID), nil
	default:
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrUnsupportedSMSProvider, cfg.Provider)
	}
}
