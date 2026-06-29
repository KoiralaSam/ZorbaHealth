package interceptors

import (
	"context"

	sharedgrpcauth "github.com/KoiralaSam/ZorbaHealth/shared/grpc/auth"
	grpcserver "google.golang.org/grpc"
)

// InternalAuthInterceptor rejects any call that doesn't carry the shared internal token.
// This ensures only trusted internal services (e.g. MCP server) can call this gRPC service.
func InternalAuthInterceptor(
	ctx context.Context,
	req any,
	info *grpcserver.UnaryServerInfo,
	handler grpcserver.UnaryHandler,
) (any, error) {
	return sharedgrpcauth.UnaryServerInterceptor(sharedgrpcauth.InternalServerConfig{
		RequireServiceID: false,
	})(ctx, req, info, handler)
}
