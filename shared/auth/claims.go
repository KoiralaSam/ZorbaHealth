package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ActorPatient = "patient"
	ActorStaff   = "staff"
	ActorAdmin   = "admin"
)

type Claims struct {
	ActorType  string
	PatientID  string
	SessionID  string
	StaffID    string
	HospitalID string
	Role       string
	AdminID    string
	Scopes     []string
}

type claimsContextKey struct{}

var errNoClaims = errors.New("no verified claims")

type patientClaims struct {
	ActorType string   `json:"actorType"`
	PatientID string   `json:"patientID"`
	SessionID string   `json:"sessionID"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

type staffClaims struct {
	ActorType  string   `json:"actorType"`
	StaffID    string   `json:"staffID"`
	HospitalID string   `json:"hospitalID"`
	Role       string   `json:"role"`
	Scopes     []string `json:"scopes"`
	jwt.RegisteredClaims
}

type adminClaims struct {
	ActorType string   `json:"actorType"`
	AdminID   string   `json:"adminID"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	if !ok || claims == nil {
		return nil, errNoClaims
	}
	return claims, nil
}

func VerifyToken(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("missing forwarded token")
	}

	parser := jwt.NewParser()
	unverified := jwt.MapClaims{}
	if _, _, err := parser.ParseUnverified(tokenStr, unverified); err != nil {
		return nil, errors.New("invalid token")
	}

	actorType, ok := unverified["actorType"].(string)
	if !ok || actorType == "" {
		return nil, errors.New("missing actorType")
	}

	secret, err := secretForActor(actorType)
	if err != nil {
		return nil, err
	}

	switch actorType {
	case ActorPatient:
		var parsed patientClaims
		if _, err := jwt.ParseWithClaims(tokenStr, &parsed, keyFunc(secret)); err != nil {
			return nil, errors.New("invalid token")
		}
		if parsed.PatientID == "" || parsed.SessionID == "" {
			return nil, errors.New("invalid patient claims")
		}
		return &Claims{
			ActorType: parsed.ActorType,
			PatientID: parsed.PatientID,
			SessionID: parsed.SessionID,
			Scopes:    parsed.Scopes,
		}, nil
	case ActorStaff:
		var parsed staffClaims
		if _, err := jwt.ParseWithClaims(tokenStr, &parsed, keyFunc(secret)); err != nil {
			return nil, errors.New("invalid token")
		}
		if parsed.StaffID == "" || parsed.HospitalID == "" {
			return nil, errors.New("invalid staff claims")
		}
		return &Claims{
			ActorType:  parsed.ActorType,
			StaffID:    parsed.StaffID,
			HospitalID: parsed.HospitalID,
			Role:       parsed.Role,
			Scopes:     parsed.Scopes,
		}, nil
	case ActorAdmin:
		var parsed adminClaims
		if _, err := jwt.ParseWithClaims(tokenStr, &parsed, keyFunc(secret)); err != nil {
			return nil, errors.New("invalid token")
		}
		if parsed.AdminID == "" {
			return nil, errors.New("invalid admin claims")
		}
		return &Claims{
			ActorType: parsed.ActorType,
			AdminID:   parsed.AdminID,
			Scopes:    parsed.Scopes,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported actorType %q", actorType)
	}
}

func RequireActorType(claims *Claims, actorType string) error {
	if claims == nil || claims.ActorType != actorType {
		return fmt.Errorf("forbidden: actor type must be %s", actorType)
	}
	return nil
}

func CheckConsent(ctx context.Context, db *pgxpool.Pool, patientID, hospitalID string) (bool, error) {
	if db == nil {
		return false, errors.New("db is required")
	}

	var allowed bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM patient_hospital_consents
			WHERE patient_id = $1
			  AND hospital_id = $2
			  AND revoked_at IS NULL
		)
	`, patientID, hospitalID).Scan(&allowed)
	return allowed, err
}

func LogAuditEventAsync(db *pgxpool.Pool, claims *Claims, tool, outcome, errorMsg string) {
	if db == nil || claims == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		actorID := claims.PatientID
		if claims.ActorType == ActorStaff {
			actorID = claims.StaffID
		}
		if claims.ActorType == ActorAdmin {
			actorID = claims.AdminID
		}

		var hospitalID any
		if claims.HospitalID != "" {
			hospitalID = claims.HospitalID
		}

		var sessionID string
		if claims.SessionID != "" {
			sessionID = claims.SessionID
		} else {
			sessionID = fmt.Sprintf("%s:%s", claims.ActorType, actorID)
		}

		var errValue any
		if errorMsg != "" {
			errValue = errorMsg
		}

		_, _ = db.Exec(ctx, `
			INSERT INTO mcp_audit_log (
				session_id,
				actor_type,
				actor_id,
				hospital_id,
				tool,
				outcome,
				error_msg
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, sessionID, claims.ActorType, actorID, hospitalID, tool, outcome, errValue)
	}()
}

func keyFunc(secret string) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid token signing method")
		}
		return []byte(secret), nil
	}
}

func secretForActor(actorType string) (string, error) {
	switch actorType {
	case ActorPatient:
		secret := os.Getenv("PATIENT_SERVICE_JWT_SECRET")
		if secret == "" {
			return "", errors.New("PATIENT_SERVICE_JWT_SECRET is not set")
		}
		return secret, nil
	case ActorStaff, ActorAdmin:
		secret := os.Getenv("AUTH_SERVICE_JWT_SECRET")
		if secret == "" {
			return "", errors.New("AUTH_SERVICE_JWT_SECRET is not set")
		}
		return secret, nil
	default:
		return "", fmt.Errorf("unsupported actorType %q", actorType)
	}
}

func HasScope(claims *Claims, required string) bool {
	if required == "" {
		return true
	}
	if claims == nil {
		return false
	}
	for _, scope := range claims.Scopes {
		if scope == required {
			return true
		}
	}
	return false
}
