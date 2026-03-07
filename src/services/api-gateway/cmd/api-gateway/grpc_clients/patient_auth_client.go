package grpc_clients

import (
	"os"

	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type patientAuthServiceClient struct {
	LoginClient      pb.LoginServiceClient
	RegistrationClient registration_verification.RegistrationVerificationServiceClient
	conn             *grpc.ClientConn
}

func NewPatientAuthServiceClient() (*patientAuthServiceClient, error) {
	patientServiceGRPCAddr := os.Getenv("PATIENT_SERVICE_GRPC_ADDR")

	if patientServiceGRPCAddr == "" {
		patientServiceGRPCAddr = "localhost:9093"
	}

	conn, err := grpc.NewClient(patientServiceGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &patientAuthServiceClient{
		LoginClient:        pb.NewLoginServiceClient(conn),
		RegistrationClient: registration_verification.NewRegistrationVerificationServiceClient(conn),
		conn:               conn,
	}, nil
}

func (c *patientAuthServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
