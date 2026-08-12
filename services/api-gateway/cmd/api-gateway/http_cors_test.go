package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	previous := gatewayCORS
	t.Cleanup(func() { gatewayCORS = previous })
	gatewayCORS = corsConfig{
		AllowedOrigins: map[string]struct{}{"https://app.zorba.test": {}},
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization",
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/patient/profile", nil)
	req.Header.Set("Origin", "https://app.zorba.test")
	rec := httptest.NewRecorder()

	corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight should not call next handler")
	})(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.zorba.test" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestCORSMiddlewareRejectsUnconfiguredPreflightOrigin(t *testing.T) {
	previous := gatewayCORS
	t.Cleanup(func() { gatewayCORS = previous })
	gatewayCORS = corsConfig{
		AllowedOrigins: map[string]struct{}{"https://app.zorba.test": {}},
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization",
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/patient/profile", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("rejected preflight should not call next handler")
	})(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSMiddlewareAllowsLocalhostOriginsWhenEnabled(t *testing.T) {
	previous := gatewayCORS
	t.Cleanup(func() { gatewayCORS = previous })
	gatewayCORS = corsConfig{
		AllowedOrigins: map[string]struct{}{"https://app.zorba.test": {}},
		AllowLocalhost: true,
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization",
	}

	for _, origin := range []string{
		"http://localhost:4173",
		"http://127.0.0.1:3999",
		"https://localhost:8443",
	} {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/patient/profile", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()

		corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("preflight should not call next handler")
		})(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("origin %q: status = %d, want %d", origin, rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("origin %q: Access-Control-Allow-Origin = %q", origin, got)
		}
	}

	// Non-localhost origins are still rejected even with the flag on.
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/patient/profile", nil)
	req.Header.Set("Origin", "http://evil.example:3000")
	rec := httptest.NewRecorder()
	corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("rejected preflight should not call next handler")
	})(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSMiddlewareRejectsLocalhostOriginsWhenDisabled(t *testing.T) {
	previous := gatewayCORS
	t.Cleanup(func() { gatewayCORS = previous })
	gatewayCORS = corsConfig{
		AllowedOrigins: map[string]struct{}{"http://localhost:3000": {}},
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization",
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/patient/profile", nil)
	req.Header.Set("Origin", "http://localhost:4173")
	rec := httptest.NewRecorder()

	corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("rejected preflight should not call next handler")
	})(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSMiddlewareAllowsSameOriginRequestsWithoutOriginHeader(t *testing.T) {
	previous := gatewayCORS
	t.Cleanup(func() { gatewayCORS = previous })
	gatewayCORS = corsConfig{
		AllowedOrigins: map[string]struct{}{"https://app.zorba.test": {}},
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patient/profile", nil)
	rec := httptest.NewRecorder()
	called := false

	corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestPatientLoginIdentifiersUsesIdentifierEmail(t *testing.T) {
	email, phone := patientLoginIdentifiers(PatientLoginRequest{
		Identifier: " PATIENT@Example.COM ",
	})

	if email != "patient@example.com" {
		t.Fatalf("email = %q", email)
	}
	if phone != "" {
		t.Fatalf("phone = %q, want empty", phone)
	}
}

func TestPatientLoginIdentifiersUsesIdentifierPhone(t *testing.T) {
	email, phone := patientLoginIdentifiers(PatientLoginRequest{
		Identifier: "+1 (555) 123-4567",
	})

	if email != "" {
		t.Fatalf("email = %q, want empty", email)
	}
	if phone != "+1 (555) 123-4567" {
		t.Fatalf("phone = %q", phone)
	}
}

func TestPatientLoginIdentifiersPreservesLegacyFields(t *testing.T) {
	email, phone := patientLoginIdentifiers(PatientLoginRequest{
		Email:       " PATIENT@Example.COM ",
		PhoneNumber: "+15551234567",
	})

	if email != "patient@example.com" {
		t.Fatalf("email = %q", email)
	}
	if phone != "+15551234567" {
		t.Fatalf("phone = %q", phone)
	}
}
