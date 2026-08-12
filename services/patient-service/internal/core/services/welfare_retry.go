package services

import (
	"time"

	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
)

const (
	defaultWelfareMaxAttempts  = 5
	defaultWelfareRetryBaseSec = 120  // 2 minutes
	defaultWelfareRetryMaxSec  = 1800 // 30 minutes
)

func welfareMaxAttempts() int32 {
	n := sharedenv.GetInt("WELFARE_CHECK_MAX_ATTEMPTS", defaultWelfareMaxAttempts)
	if n < 1 {
		return 1
	}
	if n > 20 {
		return 20
	}
	return int32(n)
}

// welfareRetryBackoff returns exponential delay after a failed attempt.
// attempt is 1-based (the attempt that just failed).
func welfareRetryBackoff(attempt int32) time.Duration {
	baseSec := sharedenv.GetInt("WELFARE_CHECK_RETRY_BASE_SECONDS", defaultWelfareRetryBaseSec)
	if baseSec < 30 {
		baseSec = 30
	}
	maxSec := sharedenv.GetInt("WELFARE_CHECK_RETRY_MAX_SECONDS", defaultWelfareRetryMaxSec)
	if maxSec < baseSec {
		maxSec = baseSec
	}

	if attempt < 1 {
		attempt = 1
	}
	delay := time.Duration(baseSec) * time.Second
	for i := int32(1); i < attempt; i++ {
		if delay >= time.Duration(maxSec)*time.Second {
			return time.Duration(maxSec) * time.Second
		}
		delay *= 2
	}
	if delay > time.Duration(maxSec)*time.Second {
		return time.Duration(maxSec) * time.Second
	}
	return delay
}

func welfareCanRetry(attempts int32) bool {
	return attempts < welfareMaxAttempts()
}
