package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// PatientJWTAuth validates WS bearer/query tokens and extracts patient identity from JWT claims.
// Expects HS256 plus actorType=patient and a patient identifier claim.
type PatientJWTAuth struct {
	secret string
}

func NewPatientJWTAuth(secret string) *PatientJWTAuth {
	return &PatientJWTAuth{secret: secret}
}

func (a *PatientJWTAuth) ExtractPatientID(token string) (string, error) {
	if a.secret == "" {
		return "", errors.New("PATIENT_SERVICE_JWT_SECRET is not set")
	}
	if token == "" {
		return "", errors.New("missing token")
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(a.secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	actorType, ok := claims["actorType"].(string)
	if !ok || actorType != "patient" {
		return "", errors.New("invalid actorType claim")
	}

	if sessionID, ok := claims["sessionID"].(string); !ok || sessionID == "" {
		return "", errors.New("sessionID claim missing")
	}

	if pid, ok := claims["patientID"].(string); ok && pid != "" {
		return pid, nil
	}
	if pid, ok := claims["patient_id"].(string); ok && pid != "" {
		return pid, nil
	}
	if pid, ok := claims["patientId"].(string); ok && pid != "" {
		return pid, nil
	}

	return "", errors.New("patient_id claim missing")
}
