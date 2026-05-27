package auth

import (
	"errors"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
)

// PatientJWTAuth validates WS bearer/query tokens and extracts patient identity from JWT claims.
type PatientJWTAuth struct{}

func NewPatientJWTAuth(_ string) *PatientJWTAuth {
	return &PatientJWTAuth{}
}

func (a *PatientJWTAuth) ExtractPatientID(token string) (string, error) {
	if token == "" {
		return "", errors.New("missing token")
	}
	// Location-service uses the same patient JWTs as the portal (shared/auth).
	claims, err := sharedauth.VerifyToken(token)
	if err != nil {
		return "", err
	}
	if claims.ActorType != sharedauth.ActorPatient {
		return "", errors.New("invalid actorType claim")
	}
	if claims.PatientID == "" {
		return "", errors.New("patient_id claim missing")
	}
	return claims.PatientID, nil
}
