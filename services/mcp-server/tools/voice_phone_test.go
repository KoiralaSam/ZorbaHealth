package tools

import (
	"testing"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestBoundVoicePhoneUsesCallerClaim(t *testing.T) {
	claims := &sharedauth.Claims{CallerPhone: "15551234567"}
	phone, err := boundVoicePhone(claims, "+1 555 999 0000")
	if err == nil {
		t.Fatalf("expected phone mismatch error")
	}
	phone, err = boundVoicePhone(claims, "")
	if err != nil || phone != "15551234567" {
		t.Fatalf("got phone=%q err=%v", phone, err)
	}
}

func TestValidateOTPFormat(t *testing.T) {
	if !validateOTPFormat("123456") {
		t.Fatal("expected valid otp")
	}
	if validateOTPFormat("12a456") || validateOTPFormat("12345") {
		t.Fatal("expected invalid otp")
	}
}

func TestConsumeVoiceVerificationInputSchemaAcceptsAuditPayload(t *testing.T) {
	schema, err := jsonschema.For[consumeVoiceVerificationInput](nil)
	if err != nil {
		t.Fatalf("schema inference failed: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("schema resolve failed: %v", err)
	}

	payload := map[string]any{
		"_auth":                      "bearer-token",
		"_verificationCorrelationId": "correlation-id",
		"verificationChannel":        "sms_poll",
	}
	if err := resolved.Validate(payload); err != nil {
		t.Fatalf("expected audit payload to validate: %v", err)
	}
}
