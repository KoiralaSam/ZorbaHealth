package interceptors

import (
	"context"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

	values := md.Get("x-forwarded-token")
	if len(values) == 0 || values[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing forwarded token")
	}

	claims, err := sharedauth.VerifyToken(values[0])
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid forwarded token")
	}

	return handler(sharedauth.WithClaims(ctx, claims), req)
}
