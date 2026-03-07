package services

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/jwt"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/outbound"
)

const bcryptCost = 10

// AuthService contains business logic for authentication flows (register, login, logout, session management, token verification).
type AuthService struct {
	userRepo outbound.UserRepository
	authRepo outbound.AuthRepository
}

func NewAuthService(userRepo outbound.UserRepository, authRepo outbound.AuthRepository) *AuthService {
	return &AuthService{userRepo: userRepo, authRepo: authRepo}
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

// Login validates credentials, creates a session (auth row + JWT), and returns the token with user_id and role.
func (s *AuthService) Login(ctx context.Context, email, phoneNumber, password string) (token string, userID, role string, err error) {
	var user *models.User
	if email != "" {
		user, err = s.userRepo.GetUserByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	} else if phoneNumber != "" {
		user, err = s.userRepo.GetUserByPhoneNumber(ctx, strings.TrimSpace(phoneNumber))
	} else {
		return "", "", "", errors.New("email or phone number required")
	}
	if err != nil {
		return "", "", "", err
	}
	if user == nil {
		return "", "", "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", "", errors.New("invalid credentials")
	}
	token, auth, err := s.CreateSession(ctx, user.ID, "")
	if err != nil {
		return "", "", "", err
	}
	return token, auth.UserID, user.Role, nil
}

// CreateSession persists an auth session and returns a signed JWT token. authUUID input is ignored; DB generates it.
func (s *AuthService) CreateSession(ctx context.Context, userID string, _ string) (string, *models.Auth, error) {
	auth, err := s.authRepo.CreateAuth(ctx, userID, "")
	if err != nil {
		return "", nil, err
	}
	token, err := jwt.GenerateToken(auth)
	if err != nil {
		return "", nil, err
	}
	return token, auth, nil
}

// Logout verifies the token, deletes the auth session, and returns success or error message.
func (s *AuthService) Logout(ctx context.Context, accessToken string) (string, error) {
	auth, err := jwt.VerifyToken(accessToken)
	if err != nil {
		return "invalid or expired token", err
	}
	if err := s.authRepo.DeleteAuth(ctx, auth.UserID, auth.AuthUUID); err != nil {
		return "failed to invalidate session", err
	}
	return "logged out successfully", nil
}

// VerifyToken parses the JWT, optionally checks the session still exists in DB, and returns claims (user_id, auth_uuid, role).
// Used by other services as "middleware as a service": call this RPC with the Bearer token to get validated claims.
func (s *AuthService) VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, err error) {
	auth, err := jwt.VerifyToken(accessToken)
	if err != nil {
		return "", "", "", err
	}
	// Optionally ensure session still exists (not logged out)
	_, err = s.authRepo.GetAuthByUserIDAndAuthUUID(ctx, auth.UserID, auth.AuthUUID)
	if err != nil {
		return "", "", "", errors.New("session not found or expired")
	}
	user, err := s.userRepo.GetUserByID(ctx, auth.UserID)
	if err != nil {
		return auth.UserID, auth.AuthUUID, "", nil // still return user_id and auth_uuid; role optional
	}
	return auth.UserID, auth.AuthUUID, user.Role, nil
}

// DeleteUser deletes a user by ID. Used by RegisterPatient to clean up on duplicate registration.
func (s *AuthService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.DeleteUser(ctx, id)
}
