package interceptors

import (
	"context"
	"os"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InternalAuthInterceptor rejects any call that doesn't carry the shared internal token.
// This ensures only trusted internal services (e.g. MCP server) can call this gRPC service.
func InternalAuthInterceptor(
	ctx context.Context,
	req any,
	_ *grpcserver.UnaryServerInfo,
	handler grpcserver.UnaryHandler,
) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	tokens := md.Get("x-internal-token")
	if len(tokens) == 0 || tokens[0] != os.Getenv("INTERNAL_SERVICE_SECRET") {
		return nil, status.Error(codes.Unauthenticated, "invalid internal token")
	}
	return handler(ctx, req)
}

