package tools

import (
	"regexp"
	"strings"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
)

var reSixDigitOTP = regexp.MustCompile(`^\d{6}$`)

func boundVoicePhone(claims *sharedauth.Claims, requested string) (string, error) {
	phone := strings.TrimSpace(requested)
	if claims != nil && claims.CallerPhone != "" {
		if phone != "" && !sharedauth.PhonesMatch(phone, claims.CallerPhone) {
			return "", errPhoneMismatch
		}
		return sharedauth.CanonicalPhoneDigits(claims.CallerPhone), nil
	}
	if phone == "" {
		return "", errPhoneRequired
	}
	canonical := sharedauth.CanonicalPhoneDigits(phone)
	if canonical == "" {
		return "", errPhoneRequired
	}
	return canonical, nil
}

func voiceAuditMeta(claims *sharedauth.Claims, channel string, verificationCorrelationID string) map[string]any {
	meta := map[string]any{}
	if claims != nil && claims.SessionID != "" {
		meta["voice_session_id"] = claims.SessionID
	}
	if channel != "" {
		meta["verification_channel"] = channel
	}
	if verificationCorrelationID != "" {
		meta["verification_correlation_id"] = verificationCorrelationID
	}
	return meta
}

func validateOTPFormat(otp string) bool {
	return reSixDigitOTP.MatchString(strings.TrimSpace(otp))
}

var (
	errPhoneMismatch = &voiceToolError{msg: "forbidden: phone number does not match caller"}
	errPhoneRequired = &voiceToolError{msg: "phone number is required"}
)

type voiceToolError struct {
	msg string
}

func (e *voiceToolError) Error() string {
	return e.msg
}
