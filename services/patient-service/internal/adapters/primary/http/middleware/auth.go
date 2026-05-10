package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type contextKey string

const (
	authDetailsKey contextKey = "authDetails"
	authClaimsKey contextKey = "authClaims"
)

// AuthClaims holds verified token claims from auth-service (middleware as a service).
type AuthClaims struct {
	UserID   string
	AuthUUID string
	Role     string
}

// TokenVerifier is implemented by the auth-service client. Pass it to AuthMiddleware to verify tokens via auth-service.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, valid bool, err error)
}

// AuthMiddleware returns a net/http middleware. If verifier is non-nil, it calls auth-service VerifyToken (middleware as a service) and sets claims on context; otherwise only checks that a token is present.
func AuthMiddleware(next http.Handler, verifier TokenVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)

		if token == "" {
			log.Printf("No token found in request from %s", r.RemoteAddr)
			writeJSONError(w, http.StatusUnauthorized, "authorization token required")
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, authDetailsKey, token)

		if verifier != nil {
			userID, authUUID, role, valid, err := verifier.VerifyToken(ctx, token)
			if err != nil {
				log.Printf("VerifyToken error from auth-service: %v", err)
				writeJSONError(w, http.StatusInternalServerError, "token verification failed")
				return
			}
			if !valid {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx = context.WithValue(ctx, authClaimsKey, &AuthClaims{UserID: userID, AuthUUID: authUUID, Role: role})
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthClaimsFromRequest returns verified claims set by AuthMiddleware when a TokenVerifier was used, or nil.
func AuthClaimsFromRequest(r *http.Request) *AuthClaims {
	v := r.Context().Value(authClaimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*AuthClaims)
	return c
}

// AuthFromRequest returns the raw token set by AuthMiddleware, or empty string if not present.
func AuthFromRequest(r *http.Request) string {
	v := r.Context().Value(authDetailsKey)
	if v == nil {
		return ""
	}
	token, _ := v.(string)
	return token
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		return authHeader
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	if c, err := r.Cookie("token"); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
