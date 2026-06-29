package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
)

// AuthService is the inbound port implemented by the core auth service.
type AuthService interface {
	RegisterUser(ctx context.Context, email, phoneNumber, password, role string) (*models.User, error)
	RegisterHospital(ctx context.Context, hospitalName, licenseNo, staffName, staffEmail, staffPhone, password, staffRole string) (*models.HospitalStaffAccount, error)
	RegisterHospitalStaff(ctx context.Context, hospitalID, staffName, staffEmail, staffPhone, password, staffRole string) (*models.HospitalStaffAccount, error)
	ValidateUserCredentials(ctx context.Context, email, phoneNumber, password string) (userID, role string, err error)
	Login(ctx context.Context, email, phoneNumber, password, clientKind string) (accessToken, refreshToken, userID, role, authUUID string, err error)
	CreateSession(ctx context.Context, userID string, authUUID string) (string, *models.Auth, error)
	CreatePatientSession(ctx context.Context, userID string, scopes []string) (accessToken, refreshToken string, auth *models.Auth, err error)
	RefreshSession(ctx context.Context, refreshPlain, expectedActorType, clientKind string) (accessToken, newRefresh, userID, authUUID, role string, err error)
	Logout(ctx context.Context, accessToken string) (string, error)
	LogoutByRefresh(ctx context.Context, refreshPlain string) error
	VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, err error)
	DeleteUser(ctx context.Context, id string) error
}
