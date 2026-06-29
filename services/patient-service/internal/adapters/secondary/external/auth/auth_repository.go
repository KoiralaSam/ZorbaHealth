package auth

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type authService struct {
	Login         pb.LoginServiceClient
	Credentials   pb.UserCredentialsServiceClient
	Register      pb.RegisterPatientServiceClient
	CreateSession pb.CreatePatientSessionServiceClient
	VerifyToken   pb.VerifyTokenServiceClient
	conn          *grpc.ClientConn
}

func NewAuthServiceClient(authServiceGRPCAddr string) (*authService, error) {
	if authServiceGRPCAddr == "" {
		authServiceGRPCAddr = "localhost:9092"
	}
	conn, err := grpcclient.DialInsecure(authServiceGRPCAddr)
	if err != nil {
		return nil, err
	}
	return &authService{
		Login:         pb.NewLoginServiceClient(conn),
		Credentials:   pb.NewUserCredentialsServiceClient(conn),
		Register:      pb.NewRegisterPatientServiceClient(conn),
		CreateSession: pb.NewCreatePatientSessionServiceClient(conn),
		VerifyToken:   pb.NewVerifyTokenServiceClient(conn),
		conn:          conn,
	}, nil
}

func (c *authService) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// authRepository implements outbound.AuthRepository using the auth gRPC client.
type authRepository struct {
	client *authService
}

// AuthRepositoryWithClose implements outbound.AuthRepository and provides Close() for the gRPC connection.
type AuthRepositoryWithClose interface {
	outbound.AuthRepository
	Close()
}

// NewAuthRepository returns an AuthRepository that calls the auth service via gRPC.
// The returned value also implements Close(); use type assertion to AuthRepositoryWithClose to close the connection.
func NewAuthRepository(authServiceGRPCAddr string) (outbound.AuthRepository, error) {
	client, err := NewAuthServiceClient(authServiceGRPCAddr)
	if err != nil {
		return nil, err
	}
	return &authRepository{client: client}, nil
}

func (r *authRepository) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResult, error) {
	if req == nil {
		return nil, nil
	}
	resp, err := r.client.Login.Login(ctx, &pb.LoginRequest{
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
	})
	if err != nil {
		return nil, err
	}
	return &models.LoginResult{
		Message:      resp.Message,
		UserID:       resp.UserId,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		Role:         resp.Role,
	}, nil
}

func (r *authRepository) ValidateUserCredentials(ctx context.Context, req *models.LoginRequest) (string, string, error) {
	if req == nil {
		return "", "", nil
	}
	resp, err := r.client.Credentials.ValidateUserCredentials(ctx, &pb.LoginRequest{
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
	})
	if err != nil {
		return "", "", err
	}
	return resp.UserId, resp.Role, nil
}

func (r *authRepository) RegisterPatient(ctx context.Context, req *models.RegisterPatientRequest) (*models.RegisterResult, error) {
	if req == nil {
		return nil, nil
	}
	var dateOfBirth *timestamppb.Timestamp
	if !req.DateOfBirth.IsZero() {
		dateOfBirth = timestamppb.New(req.DateOfBirth)
	}
	resp, err := r.client.Register.RegisterPatient(ctx, &pb.RegisterPatientRequest{
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Password:    req.Password,
		FullName:    req.FullName,
		DateOfBirth: dateOfBirth,
	})
	if err != nil {
		return nil, err
	}
	return &models.RegisterResult{
		Message: resp.Message,
		UserID:  resp.UserId,
	}, nil
}

func (r *authRepository) CreatePatientSession(ctx context.Context, userID string, scopes []string) (*models.LoginResult, error) {
	resp, err := r.client.CreateSession.CreatePatientSession(ctx, &pb.CreatePatientSessionRequest{
		UserId: userID,
		Scopes: scopes,
	})
	if err != nil {
		return nil, err
	}
	return &models.LoginResult{
		Message:      resp.Message,
		UserID:       resp.UserId,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (r *authRepository) Close() {
	r.client.Close()
}

// TokenVerifier is implemented by authRepository so HTTP middleware can verify tokens via auth-service (middleware as a service).
func (r *authRepository) VerifyToken(ctx context.Context, accessToken string) (userID, authUUID, role string, valid bool, err error) {
	resp, err := r.client.VerifyToken.VerifyToken(ctx, &pb.VerifyTokenRequest{AccessToken: accessToken})
	if err != nil {
		return "", "", "", false, err
	}
	if resp == nil || !resp.Valid {
		return "", "", "", false, nil
	}
	return resp.UserId, resp.AuthUuid, resp.Role, true, nil
}
