package auth

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	"google.golang.org/grpc"
)

var _ outbound.AuthRepository = (*Repository)(nil)

type Repository struct {
	register pb.RegisterHealthProviderServiceClient
	conn     *grpc.ClientConn
}

func NewRepository(authServiceGRPCAddr string) (*Repository, error) {
	if authServiceGRPCAddr == "" {
		authServiceGRPCAddr = "localhost:9092"
	}
	conn, err := grpcclient.DialInsecure(authServiceGRPCAddr)
	if err != nil {
		return nil, err
	}
	return &Repository{
		register: pb.NewRegisterHealthProviderServiceClient(conn),
		conn:     conn,
	}, nil
}

func (r *Repository) Close() {
	if r.conn != nil {
		_ = r.conn.Close()
	}
}

func (r *Repository) RegisterHospitalStaffUser(ctx context.Context, email, phoneNumber, password string) (string, error) {
	resp, err := r.register.RegisterHealthProvider(ctx, &pb.RegisterHealthProviderRequest{
		Email:       email,
		PhoneNumber: phoneNumber,
		Password:    password,
	})
	if err != nil {
		return "", err
	}
	return resp.GetUserId(), nil
}
