package interceptors

import (
	"context"
	"crypto/subtle"
	"os"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

	values := md.Get("x-internal-token")
	if len(values) == 0 || values[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing internal token")
	}

	expected := os.Getenv("INTERNAL_SERVICE_SECRET")
	if expected == "" {
		return nil, status.Error(codes.Unauthenticated, "internal auth misconfigured")
	}

	if subtle.ConstantTimeCompare([]byte(values[0]), []byte(expected)) != 1 {
		return nil, status.Error(codes.Unauthenticated, "invalid internal token")
	}

	return handler(ctx, req)
}
