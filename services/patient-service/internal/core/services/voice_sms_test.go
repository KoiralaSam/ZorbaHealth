package services

import "testing"

func TestExtractVoiceOTP(t *testing.T) {
	cases := []struct {
		body string
		want string
		ok   bool
	}{
		{"123456", "123456", true},
		{" 123456 ", "123456", true},
		{"Your code is 123456 thanks", "123456", true},
		{"ignore 1234567", "", false},
		{"no digits", "", false},
	}
	for _, tc := range cases {
		got, ok := extractVoiceOTP(tc.body)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("extractVoiceOTP(%q) = %q, %v; want %q, %v", tc.body, got, ok, tc.want, tc.ok)
		}
	}
}
