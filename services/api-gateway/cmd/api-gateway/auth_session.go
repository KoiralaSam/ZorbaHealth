package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

const (
	cookiePathPatientRefresh  = "/api/v1/auth/patient"
	cookiePathHospitalRefresh = "/api/v1/auth/hospital"
	cookieNamePatientRefresh  = "zorba_patient_refresh"
	cookieNameHospitalRefresh = "zorba_staff_refresh"
)

func refreshCookieMaxAge() int {
	d, err := time.ParseDuration(env.GetString("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		d = 7 * 24 * time.Hour
	}
	return int(d.Seconds())
}

func clientKindFromRequest(r *http.Request) string {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Zorba-Client")), "mobile") {
		return "mobile"
	}
	return "web"
}

func setRefreshCookie(w http.ResponseWriter, name, value, path string) {
	secure := env.GetString("ENVIRONMENT", "development") != "development"
	sameSite := http.SameSiteNoneMode
	if !secure {
		sameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   refreshCookieMaxAge(),
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func clearRefreshCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
}

func readRefreshCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil || c == nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func refreshTokenFromRequest(r *http.Request, cookieName, bodyToken string) string {
	if t := readRefreshCookie(r, cookieName); t != "" {
		return t
	}
	if bodyToken != "" {
		return bodyToken
	}
	auth := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if auth != "" && !strings.Contains(auth, ".") {
		return auth
	}
	return ""
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	if host != "" {
		return host
	}
	return r.RemoteAddr
}
