package patientservice

import (
	"context"
	"fmt"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	conn *grpc.ClientConn
	rpc  registration_verification.RegistrationVerificationServiceClient
}

var _ outbound.PatientIdentityClient = (*Client)(nil)

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("patient-service dial failed: %w", err)
	}
	return &Client{
		conn: conn,
		rpc:  registration_verification.NewRegistrationVerificationServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) LookupByPhone(ctx context.Context, phone string) (*models.PatientCandidate, error) {
	resp, err := c.rpc.LookupPatientByPhone(ctx, &registration_verification.LookupPatientByPhoneRequest{
		PhoneNumber: phone,
	})
	if err != nil {
		return nil, fmt.Errorf("lookup patient by phone: %w", err)
	}
	if resp == nil || !resp.Found || resp.PatientId == "" {
		return nil, nil
	}
	return &models.PatientCandidate{
		PatientID: resp.PatientId,
		FullName:  resp.FullName,
	}, nil
}

func (c *Client) StartExistingPhoneVerification(ctx context.Context, phone string) error {
	_, err := c.rpc.StartExistingPhoneVerification(ctx, &registration_verification.StartExistingPhoneVerificationRequest{
		PhoneNumber: phone,
	})
	if err != nil {
		return fmt.Errorf("start existing phone verification: %w", err)
	}
	return nil
}

func (c *Client) VerifyExistingPhoneOTP(ctx context.Context, phone, otp string) (*models.IdentifiedPatient, error) {
	resp, err := c.rpc.VerifyExistingPhoneOTP(ctx, &registration_verification.VerifyExistingPhoneOTPRequest{
		PhoneNumber: phone,
		Otp:         otp,
	})
	if err != nil {
		return nil, fmt.Errorf("verify existing phone otp: %w", err)
	}
	if resp == nil || resp.PatientId == "" {
		return nil, fmt.Errorf("verify existing phone otp: empty patient id")
	}
	return &models.IdentifiedPatient{PatientID: resp.PatientId}, nil
}

func (c *Client) StartRegistration(ctx context.Context, req models.RegistrationRequest) (string, error) {
	resp, err := c.rpc.StartRegistration(ctx, &registration_verification.StartRegistrationRequest{
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		FullName:    req.FullName,
		DateOfBirth: timestamppb.New(req.DateOfBirth),
	})
	if err != nil {
		return "", fmt.Errorf("start registration: %w", err)
	}
	if resp == nil || resp.RegistrationToken == "" {
		return "", fmt.Errorf("start registration: empty registration token")
	}
	return resp.RegistrationToken, nil
}

func (c *Client) VerifyRegistrationOTPAndCreatePatient(ctx context.Context, phone, otp, registrationToken string) (*models.IdentifiedPatient, error) {
	if _, err := c.rpc.VerifyPhoneOTP(ctx, &registration_verification.VerifyPhoneOTPRequest{
		PhoneNumber: phone,
		Otp:         otp,
	}); err != nil {
		return nil, fmt.Errorf("verify registration phone otp: %w", err)
	}
	resp, err := c.rpc.CompletePhoneRegistration(ctx, &registration_verification.CompletePhoneRegistrationRequest{
		RegistrationToken: registrationToken,
	})
	if err != nil {
		return nil, fmt.Errorf("complete phone registration: %w", err)
	}
	if resp == nil || resp.PatientId == "" {
		return nil, fmt.Errorf("complete phone registration: empty patient id")
	}
	return &models.IdentifiedPatient{PatientID: resp.PatientId}, nil
}
