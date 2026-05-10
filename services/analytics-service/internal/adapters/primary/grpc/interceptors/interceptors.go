package interceptors

import (
	"context"
	"os"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
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

	tokens := md.Get("x-internal-token")
	if len(tokens) == 0 || tokens[0] != os.Getenv("INTERNAL_SERVICE_SECRET") {
		return nil, status.Error(codes.Unauthenticated, "invalid internal token")
	}

	return handler(ctx, req)
}

func ClaimsInterceptor(
	ctx context.Context,
	req any,
	_ *grpcserver.UnaryServerInfo,
	handler grpcserver.UnaryHandler,
) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("x-forwarded-token")
	if len(tokens) == 0 || tokens[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing forwarded token")
	}

	claims, err := sharedauth.VerifyToken(tokens[0])
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid forwarded token")
	}

	return handler(sharedauth.WithClaims(ctx, claims), req)
}
