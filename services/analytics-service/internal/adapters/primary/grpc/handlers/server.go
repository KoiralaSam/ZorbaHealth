package grpc

import (
	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/adapters/primary/grpc/interceptors"
	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/ports/inbound"
	"github.com/jackc/pgx/v5/pgxpool"
	grpcserver "google.golang.org/grpc"
)

func NewServer(db *pgxpool.Pool, svc inbound.AnalyticsService) *grpcserver.Server {
	server := grpcserver.NewServer(
		grpcserver.ChainUnaryInterceptor(
			interceptors.InternalAuthInterceptor,
			interceptors.ClaimsInterceptor,
		),
	)
	NewAnalyticsGRPCHandler(server, svc, db)
	return server
}
