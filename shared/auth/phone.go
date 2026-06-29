package auth

// NormalizePhoneDigits keeps only ASCII digits for consistent voice/SMS binding.
func NormalizePhoneDigits(phone string) string {
	var b []byte
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b = append(b, byte(r))
		}
	}
	return string(b)
}

// PhonesMatch returns true when both normalize to the same non-empty digit string.
func PhonesMatch(a, b string) bool {
	na := NormalizePhoneDigits(a)
	nb := NormalizePhoneDigits(b)
	return na != "" && na == nb
}
