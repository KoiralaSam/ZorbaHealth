package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
)

type PatientSessionClaims struct {
	ActorType string   `json:"actorType"`
	PatientID string   `json:"patientID"`
	SessionID string   `json:"sessionID"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

type SessionTokenIssuer struct{}

var _ outbound.SessionTokenIssuer = (*SessionTokenIssuer)(nil)

func NewSessionTokenIssuer() *SessionTokenIssuer {
	return &SessionTokenIssuer{}
}

func (s *SessionTokenIssuer) MintSessionToken(patientID, sessionID string, scopes []string) (string, error) {
	if patientID == "" {
		return "", errors.New("patientID is required")
	}
	if sessionID == "" {
		return "", errors.New("sessionID is required")
	}

	secret := os.Getenv("PATIENT_SERVICE_JWT_SECRET")
	if secret == "" {
		return "", errors.New("PATIENT_SERVICE_JWT_SECRET is not set")
	}

	now := time.Now()
	if len(scopes) == 0 {
		// Default to a restricted set: emergency + location only (no records access).
		scopes = []string{"location:read", "emergency:write"}
	}
	claims := PatientSessionClaims{
		ActorType: "patient",
		PatientID: patientID,
		SessionID: sessionID,
		Scopes:    scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   patientID,
			ExpiresAt: jwt.NewNumericDate(now.Add(3 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "zorba-agent-worker",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
