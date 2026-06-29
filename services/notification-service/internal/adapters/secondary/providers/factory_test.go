package providers

import "testing"

func TestNewEmailProviderDefaultsToMailtrap(t *testing.T) {
	provider, err := NewEmailProvider(EmailFactoryConfig{
		Provider:  "",
		APIToken:  "token",
		FromEmail: "test@example.com",
		FromName:  "Zorba",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := provider.ProviderName(); got != "mailtrap" {
		t.Fatalf("expected mailtrap provider, got %q", got)
	}
}

func TestNewSMSProviderDefaultsToVoipms(t *testing.T) {
	provider, err := NewSMSProvider(SMSFactoryConfig{
		Provider: "",
		DID:      "15551234567",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := provider.ProviderName(); got != "voipms" {
		t.Fatalf("expected voipms provider, got %q", got)
	}
}

func TestNewEmailProviderRejectsUnknownProvider(t *testing.T) {
	if _, err := NewEmailProvider(EmailFactoryConfig{Provider: "ses"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestNewSMSProviderRejectsUnknownProvider(t *testing.T) {
	if _, err := NewSMSProvider(SMSFactoryConfig{Provider: "twilio"}); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
