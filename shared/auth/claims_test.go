package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyTokenAcceptsBearerPrefixedStaffToken(t *testing.T) {
	const secret = "test-staff-secret"
	t.Setenv("AUTH_SERVICE_JWT_SECRET", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"actorType":  "staff",
		"staffID":    "staff-1",
		"hospitalID": "hospital-1",
		"role":       "doctor",
		"scopes":     []string{"patient:read"},
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := VerifyToken("Bearer " + signed)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.ActorType != ActorStaff || claims.StaffID != "staff-1" || claims.HospitalID != "hospital-1" {
		t.Fatalf("claims = %#v", claims)
	}
}

func TestNormalizeBearerToken(t *testing.T) {
	if got := NormalizeBearerToken("  bearer token-value  "); got != "token-value" {
		t.Fatalf("NormalizeBearerToken() = %q, want token-value", got)
	}
	if got := NormalizeBearerToken("token-value"); got != "token-value" {
		t.Fatalf("NormalizeBearerToken() = %q, want token-value", got)
	}
	if got := NormalizeBearerToken("Bearer token value"); got != "Bearer token value" {
		t.Fatalf("NormalizeBearerToken() = %q, want original malformed value", got)
	}
}
