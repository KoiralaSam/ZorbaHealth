package grpc_clients

import (
	"os"

	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	healthproviderpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_provider"
	"google.golang.org/grpc"
)

type healthProviderServiceClient struct {
	RegisterClient healthproviderpb.HealthProviderServiceClient
	conn           *grpc.ClientConn
}

func NewHealthProviderServiceClient() (*healthProviderServiceClient, error) {
	addr := os.Getenv("HEALTH_PROVIDER_SERVICE_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:9094"
	}
	conn, err := grpcclient.DialInsecure(addr)
	if err != nil {
		return nil, err
	}
	return &healthProviderServiceClient{
		RegisterClient: healthproviderpb.NewHealthProviderServiceClient(conn),
		conn:           conn,
	}, nil
}

func (c *healthProviderServiceClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
