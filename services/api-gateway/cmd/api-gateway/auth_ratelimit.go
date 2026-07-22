package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	"github.com/KoiralaSam/ZorbaHealth/shared/ratelimit"
)

var gatewayRateLimiter = ratelimit.NewFromEnv()

func allowOTPVerifyRate(r *http.Request, phone string) bool {
	window := envDurationSeconds("AUTH_OTP_VERIFY_WINDOW_SEC", 15*time.Minute)
	if phone != "" {
		key := "otp:verify:phone:" + sharedlogging.HashIdentifier(phone)
		ok, _, _ := gatewayRateLimiter.Allow(r.Context(), key, envInt("AUTH_OTP_VERIFY_LIMIT_PER_PHONE", 5), window)
		if !ok {
			return false
		}
	}
	ipKey := "otp:verify:ip:" + sharedlogging.HashIdentifier(clientIP(r))
	ok, _, _ := gatewayRateLimiter.Allow(r.Context(), ipKey, envInt("AUTH_OTP_VERIFY_LIMIT_PER_IP", 20), window)
	return ok
}

func recordOTPVerifyFailure(r *http.Request, phone string) {
	if phone == "" {
		return
	}
	key := "otp:verify:fail:phone:" + sharedlogging.HashIdentifier(phone)
	_, _, _ = gatewayRateLimiter.Allow(
		r.Context(),
		key,
		envInt("AUTH_OTP_VERIFY_LIMIT_PER_PHONE", 5),
		envDurationSeconds("AUTH_OTP_VERIFY_WINDOW_SEC", 15*time.Minute),
	)
}

func allowOTPSendRate(r *http.Request, phone string) bool {
	key := "otp:send:phone:" + sharedlogging.HashIdentifier(phone)
	ok, _, _ := gatewayRateLimiter.Allow(
		r.Context(),
		key,
		envInt("AUTH_OTP_SEND_LIMIT_PER_PHONE", 3),
		envDurationSeconds("AUTH_OTP_SEND_WINDOW_SEC", time.Hour),
	)
	return ok
}

func envInt(key string, def int) int {
	v := env.GetString(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDurationSeconds(key string, def time.Duration) time.Duration {
	v := env.GetString(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}
