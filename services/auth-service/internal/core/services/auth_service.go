package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/jwt"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

const bcryptCost = 10
const hospitalStaffUserRole = "hospital_staff"

// AuthService contains business logic for authentication flows (register, login, logout, session management, token verification).
type AuthService struct {
	userRepo    outbound.UserRepository
	authRepo    outbound.AuthRepository
	refreshRepo outbound.RefreshRepository
}

func NewAuthService(userRepo outbound.UserRepository, authRepo outbound.AuthRepository, refreshRepo outbound.RefreshRepository) *AuthService {
	return &AuthService{userRepo: userRepo, authRepo: authRepo, refreshRepo: refreshRepo}
}

func refreshTokenTTL() time.Duration {
	if s := env.GetString("REFRESH_TOKEN_TTL", "168h"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 7 * 24 * time.Hour
}

// RegisterUser creates a user with hashed password and returns the user (with ID). Used by RegisterPatient / RegisterHealthProvider.
func (s *AuthService) RegisterUser(ctx context.Context, email, phoneNumber, password, role string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" && strings.TrimSpace(phoneNumber) == "" {
		return nil, errors.New("email or phone number required")
	}
	if role == "" {
		role = "patient"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Email:        email,
		PhoneNumber:  strings.TrimSpace(phoneNumber),
		PasswordHash: string(hash),
		Role:         role,
	}
	return s.userRepo.CreateUser(ctx, user)
}

func (s *AuthService) RegisterHospital(ctx context.Context, hospitalName, licenseNo, staffName, staffEmail, staffPhone, password, staffRole string) (*models.HospitalStaffAccount, error) {
	hospitalName = strings.TrimSpace(hospitalName)
	licenseNo = strings.TrimSpace(licenseNo)
	staffName = strings.TrimSpace(staffName)
	staffEmail = strings.TrimSpace(strings.ToLower(staffEmail))
	staffPhone = strings.TrimSpace(staffPhone)
	staffRole = normalizeStaffRole(staffRole)
	if hospitalName == "" || licenseNo == "" || staffName == "" || staffEmail == "" || password == "" {
		return nil, errors.New("hospital_name, license_no, staff_name, email, and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	return s.userRepo.RegisterHospitalWithStaff(ctx, models.HospitalRegistration{
		HospitalName: hospitalName,
		LicenseNo:    licenseNo,
		StaffName:    staffName,
		StaffEmail:   staffEmail,
		StaffPhone:   staffPhone,
		StaffRole:    staffRole,
		PasswordHash: string(hash),
	})
}

func (s *AuthService) RegisterHospitalStaff(ctx context.Context, hospitalID, staffName, staffEmail, staffPhone, password, staffRole string) (*models.HospitalStaffAccount, error) {
	hospitalID = strings.TrimSpace(hospitalID)
	staffName = strings.TrimSpace(staffName)
	staffEmail = strings.TrimSpace(strings.ToLower(staffEmail))
	staffPhone = strings.TrimSpace(staffPhone)
	staffRole = normalizeStaffRole(staffRole)
	if hospitalID == "" || staffName == "" || staffEmail == "" || password == "" {
		return nil, errors.New("hospital_id, staff_name, email, and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	return s.userRepo.RegisterStaff(ctx, models.HospitalStaffRegistration{
		HospitalID:   hospitalID,
		StaffName:    staffName,
		StaffEmail:   staffEmail,
		StaffPhone:   staffPhone,
		StaffRole:    staffRole,
		PasswordHash: string(hash),
	})
}

func normalizeStaffRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "doctor", "nurse", "billing":
		return strings.TrimSpace(strings.ToLower(role))
	default:
		return "admin"
	}
}

// ValidateUserCredentials checks password without creating a session.
func (s *AuthService) ValidateUserCredentials(ctx context.Context, email, phoneNumber, password string) (userID, role string, err error) {
	user, err := s.lookupUser(ctx, email, phoneNumber)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid credentials")
	}
	return user.ID, user.Role, nil
}

// Login validates credentials, creates a session (auth row + JWT + refresh), and returns tokens with user_id and role.
func (s *AuthService) Login(ctx context.Context, email, phoneNumber, password, clientKind string) (accessToken, refreshToken, userID, role string, authUUID string, err error) {
	user, err := s.lookupUser(ctx, email, phoneNumber)
	if err != nil {
		return "", "", "", "", "", err
	}
	if user == nil {
		return "", "", "", "", "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", "", "", "", errors.New("invalid credentials")
	}
	accessToken, refreshToken, auth, err := s.createSessionWithRefresh(ctx, user.ID, clientKind, nil)
	if err != nil {
		return "", "", "", "", "", err
	}
	return accessToken, refreshToken, auth.UserID, user.Role, auth.AuthUUID, nil
}

// CreateSession persists an auth session and returns a signed JWT token. authUUID input is ignored; DB generates it.
func (s *AuthService) CreateSession(ctx context.Context, userID string, _ string) (string, *models.Auth, error) {
	access, _, auth, err := s.createSessionWithRefresh(ctx, userID, "", nil)
	return access, auth, err
}

func (s *AuthService) CreatePatientSession(ctx context.Context, userID string, scopes []string) (accessToken, refreshToken string, auth *models.Auth, err error) {
	return s.createSessionWithRefresh(ctx, userID, "", scopes)
}

func (s *AuthService) createSessionWithRefresh(ctx context.Context, userID, clientKind string, scopes []string) (accessToken, refreshToken string, auth *models.Auth, err error) {
	auth, err = s.authRepo.CreateAuth(ctx, userID, "")
	if err != nil {
		return "", "", nil, err
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", nil, err
	}
	actor, err := s.userRepo.ResolveSessionActor(ctx, user.ID, user.Role, auth.AuthUUID)
	if err != nil {
		return "", "", nil, err
	}
	if len(scopes) > 0 {
		actor.Scopes = scopes
	}
	accessToken, err = jwt.GenerateToken(auth, actor)
	if err != nil {
		return "", "", nil, err
	}
	plain, hash, err := newOpaqueRefreshToken()
	if err != nil {
		return "", "", nil, err
	}
	expires := time.Now().Add(refreshTokenTTL())
	if err := s.refreshRepo.InsertRefresh(ctx, auth.AuthUUID, userID, actor.ActorType, hash, 1, expires); err != nil {
		return "", "", nil, err
	}
	return accessToken, plain, auth, nil
}

// RefreshSession rotates refresh token and returns new access + refresh tokens.
func (s *AuthService) RefreshSession(ctx context.Context, refreshPlain, expectedActorType, clientKind string) (accessToken, newRefresh, userID, authUUID, role string, err error) {
	hash, err := hashRefreshToken(refreshPlain)
	if err != nil {
		return "", "", "", "", "", domainerrors.ErrInvalidRefreshToken
	}
	newPlain, newHash, err := newOpaqueRefreshToken()
	if err != nil {
		return "", "", "", "", "", err
	}
	expires := time.Now().Add(refreshTokenTTL())
	_, authUUID, userID, actorType, reuse, err := s.refreshRepo.RotateRefresh(ctx, hash, newHash, expires)
	if reuse {
		return "", "", userID, authUUID, "", domainerrors.ErrRefreshTokenReuse
	}
	if err != nil {
		return "", "", "", "", "", err
	}
	if expectedActorType != "" && actorType != expectedActorType {
		return "", "", "", "", "", domainerrors.ErrInvalidRefreshToken
	}
	auth, err := s.authRepo.GetAuthByAuthUUID(ctx, authUUID)
	if err != nil || auth.RevokedAt != nil {
		return "", "", "", "", "", domainerrors.ErrSessionRevoked
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", "", "", "", err
	}
	actor, err := s.userRepo.ResolveSessionActor(ctx, user.ID, user.Role, auth.AuthUUID)
	if err != nil {
		return "", "", "", "", "", err
	}
	accessToken, err = jwt.GenerateToken(auth, actor)
	if err != nil {
		return "", "", "", "", "", err
	}
	return accessToken, newPlain, userID, authUUID, user.Role, nil
}

// LogoutByRefresh revokes session family using refresh token.
func (s *AuthService) LogoutByRefresh(ctx context.Context, refreshPlain string) error {
	hash, err := hashRefreshToken(refreshPlain)
	if err != nil {
		return domainerrors.ErrInvalidRefreshToken
	}
	authUUID, err := s.refreshRepo.GetAuthUUIDByTokenHash(ctx, hash)
	if err != nil {
		return err
	}
	if err := s.refreshRepo.RevokeFamily(ctx, authUUID, "logout"); err != nil {
		return err
	}
	return s.authRepo.RevokeAuth(ctx, authUUID, "logout")
}

// Logout verifies the token, revokes the auth session family, and returns success or error message.
func (s *AuthService) Logout(ctx context.Context, accessToken string) (string, error) {
	authClaims, err := jwt.VerifyToken(accessToken)
	if err != nil {
		return "invalid or expired token", err
	}
	if err := s.refreshRepo.RevokeFamily(ctx, authClaims.AuthUUID, "logout"); err != nil {
		return "failed to invalidate session", err
	}
	if err := s.authRepo.RevokeAuth(ctx, authClaims.AuthUUID, "logout"); err != nil {
		return "failed to invalidate session", err
	}
	return "logged out successfully", nil
}

// VerifyToken parses the JWT, checks the session still exists and is not revoked, and returns claims.
func (s *AuthService) VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, err error) {
	authClaims, err := jwt.VerifyToken(accessToken)
	if err != nil {
		return "", "", "", err
	}
	auth, err := s.authRepo.GetAuthByUserIDAndAuthUUID(ctx, authClaims.UserID, authClaims.AuthUUID)
	if err != nil {
		return "", "", "", errors.New("session not found or expired")
	}
	if auth.RevokedAt != nil {
		return "", "", "", errors.New("session not found or expired")
	}
	user, err := s.userRepo.GetUserByID(ctx, authClaims.UserID)
	if err != nil {
		return authClaims.UserID, authClaims.AuthUUID, "", nil
	}
	return authClaims.UserID, authClaims.AuthUUID, user.Role, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *AuthService) lookupUser(ctx context.Context, email, phoneNumber string) (*models.User, error) {
	if email != "" {
		return s.userRepo.GetUserByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	}
	if phoneNumber != "" {
		return s.userRepo.GetUserByPhoneNumber(ctx, strings.TrimSpace(phoneNumber))
	}
	return nil, errors.New("email or phone number required")
}
