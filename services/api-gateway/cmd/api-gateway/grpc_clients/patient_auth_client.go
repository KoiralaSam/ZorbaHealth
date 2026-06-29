package grpc_clients

import (
	"os"

	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	authpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	"google.golang.org/grpc"
)

type patientAuthServiceClient struct {
	LoginClient            pb.LoginServiceClient
	RegistrationClient     registration_verification.RegistrationVerificationServiceClient
	HospitalLoginClient    authpb.LoginServiceClient
	HospitalRegisterClient authpb.RegisterHealthProviderServiceClient
	RefreshClient          authpb.RefreshSessionServiceClient
	LogoutRefreshClient    authpb.LogoutServiceClient
	patientConn            *grpc.ClientConn
	authConn               *grpc.ClientConn
}

func NewPatientAuthServiceClient() (*patientAuthServiceClient, error) {
	patientServiceGRPCAddr := os.Getenv("PATIENT_SERVICE_GRPC_ADDR")
	authServiceGRPCAddr := os.Getenv("AUTH_SERVICE_GRPC_ADDR")

	if patientServiceGRPCAddr == "" {
		patientServiceGRPCAddr = "localhost:9093"
	}
	if authServiceGRPCAddr == "" {
		authServiceGRPCAddr = "localhost:9092"
	}

	patientConn, err := grpcclient.DialInsecure(patientServiceGRPCAddr)
	if err != nil {
		return nil, err
	}
	authConn, err := grpcclient.DialInsecure(authServiceGRPCAddr)
	if err != nil {
		_ = patientConn.Close()
		return nil, err
	}

	return &patientAuthServiceClient{
		LoginClient:            pb.NewLoginServiceClient(patientConn),
		RegistrationClient:     registration_verification.NewRegistrationVerificationServiceClient(patientConn),
		HospitalLoginClient:    authpb.NewLoginServiceClient(authConn),
		HospitalRegisterClient: authpb.NewRegisterHealthProviderServiceClient(authConn),
		RefreshClient:          authpb.NewRefreshSessionServiceClient(authConn),
		LogoutRefreshClient:    authpb.NewLogoutServiceClient(authConn),
		patientConn:            patientConn,
		authConn:               authConn,
	}, nil
}

func (c *patientAuthServiceClient) Close() {
	if c.patientConn != nil {
		_ = c.patientConn.Close()
	}
	if c.authConn != nil {
		_ = c.authConn.Close()
	}
}
