package jwt

import (
	"errors"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/golang-jwt/jwt/v5"
)

// Auth is a convenience alias for a pointer to the domain Auth model.
type Auth = *models.Auth

type SessionActor = *models.SessionActor

// GenerateToken creates a signed JWT for the given auth/session.
func GenerateToken(claims Auth, actor SessionActor) (string, error) {
	secret := env.GetString("AUTH_SERVICE_JWT_SECRET", "")
	if actor == nil {
		return "", errors.New("session actor required")
	}

	tokenClaims := jwt.MapClaims{
		"authorized": true,
		"auth_uuid":  claims.AuthUUID,
		"user_id":    claims.UserID,
		"actorType":  actor.ActorType,
		"scopes":     actor.Scopes,
		"iat":        time.Now().Unix(),
	}

	switch actor.ActorType {
	case "patient":
		tokenClaims["patientID"] = actor.PatientID
		tokenClaims["sessionID"] = actor.SessionID
	case "staff":
		tokenClaims["staffID"] = actor.StaffID
		tokenClaims["hospitalID"] = actor.HospitalID
		tokenClaims["role"] = actor.Role
	case "admin":
		tokenClaims["adminID"] = actor.AdminID
	default:
		return "", errors.New("unsupported actor type")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)

	return token.SignedString([]byte(secret))
}

// VerifyToken parses and validates a JWT and returns the associated Auth claims.
func VerifyToken(token string) (*models.Auth, error) {
	secret := env.GetString("AUTH_SERVICE_JWT_SECRET", "")
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("invalid token signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, errors.New("invalid token")
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("could not extract userId from token")
	}

	authUUID, ok := claims["auth_uuid"].(string)
	if !ok {
		return nil, errors.New("could not extract uuid from token")
	}

	authDetails := &models.Auth{
		UserID:   userID,
		AuthUUID: authUUID,
	}

	return authDetails, nil
}
