package auth

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidPhoneNumber is returned when a phone cannot be stored or matched.
	ErrInvalidPhoneNumber = errors.New("invalid phone number: must be 10–15 digits (optional leading +); NANP numbers are stored as 11 digits with country code 1")

	// Strict input: optional +, then 10–15 digits. Separators are rejected on write.
	phoneInputPattern = regexp.MustCompile(`^\+?[0-9]{10,15}$`)
)

// NormalizePhoneDigits keeps only ASCII digits (no country-code canonicalization).
func NormalizePhoneDigits(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CanonicalPhoneDigits returns the storage form: digits-only E.164 without '+'.
// US/CA NANP numbers are always stored as 11 digits with leading country code 1:
//   3185125670  → 13185125670
//   13185125670 → 13185125670
// Non-NANP international numbers (other lengths / non-1 country codes) are stored as digits.
func CanonicalPhoneDigits(phone string) string {
	digits := NormalizePhoneDigits(phone)
	switch {
	case digits == "":
		return ""
	case len(digits) == 10:
		return "1" + digits
	case len(digits) == 11 && digits[0] == '1':
		return digits
	default:
		return digits
	}
}

// ValidatePhoneForStorage enforces the strict input pattern and returns the
// canonical digits-only form used in users.phone_number / patients.phone_number.
func ValidatePhoneForStorage(phone string) (string, error) {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" || !phoneInputPattern.MatchString(trimmed) {
		return "", ErrInvalidPhoneNumber
	}

	digits := NormalizePhoneDigits(trimmed)
	switch {
	case len(digits) == 10:
		// National NANP must not start with 0/1 (those are not valid area-code leads).
		if digits[0] == '0' || digits[0] == '1' {
			return "", ErrInvalidPhoneNumber
		}
	case len(digits) == 11 && digits[0] == '1':
		if digits[1] == '0' || digits[1] == '1' {
			return "", ErrInvalidPhoneNumber
		}
	case len(digits) < 10 || len(digits) > 15:
		return "", ErrInvalidPhoneNumber
	}

	canonical := CanonicalPhoneDigits(trimmed)
	if canonical == "" || len(canonical) < 10 || len(canonical) > 15 {
		return "", ErrInvalidPhoneNumber
	}
	return canonical, nil
}

// PhonesMatch returns true when both numbers canonicalize to the same non-empty digit string.
// This equates SIP 10-digit NANP Caller-ID with stored 11-digit E.164 forms.
func PhonesMatch(a, b string) bool {
	ca := CanonicalPhoneDigits(a)
	cb := CanonicalPhoneDigits(b)
	return ca != "" && ca == cb
}
