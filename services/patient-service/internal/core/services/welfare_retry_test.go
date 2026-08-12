package services

import (
	"testing"
	"time"
)

func TestWelfareRetryBackoffExponential(t *testing.T) {
	t.Setenv("WELFARE_CHECK_RETRY_BASE_SECONDS", "120")
	t.Setenv("WELFARE_CHECK_RETRY_MAX_SECONDS", "1800")

	cases := []struct {
		attempt int32
		want    time.Duration
	}{
		{1, 2 * time.Minute},
		{2, 4 * time.Minute},
		{3, 8 * time.Minute},
		{4, 16 * time.Minute},
		{5, 30 * time.Minute}, // capped
		{6, 30 * time.Minute},
	}
	for _, tc := range cases {
		got := welfareRetryBackoff(tc.attempt)
		if got != tc.want {
			t.Fatalf("attempt %d: got %v want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestWelfareCanRetryRespectsMaxAttempts(t *testing.T) {
	t.Setenv("WELFARE_CHECK_MAX_ATTEMPTS", "5")
	if !welfareCanRetry(4) {
		t.Fatal("expected retry when attempts < max")
	}
	if welfareCanRetry(5) {
		t.Fatal("expected no retry when attempts == max")
	}
}
