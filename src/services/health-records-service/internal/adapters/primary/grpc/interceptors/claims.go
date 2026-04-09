package interceptors

import (
	"context"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ClaimsInterceptor verifies the forwarded end-user JWT and injects claims into the request context.
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

// Chain enforces trusted-service auth first, then end-user forwarded claims.
func Chain() grpcserver.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpcserver.UnaryServerInfo,
		handler grpcserver.UnaryHandler,
	) (any, error) {
		return InternalAuthInterceptor(ctx, req, info, func(ctx context.Context, req any) (any, error) {
			return ClaimsInterceptor(ctx, req, info, handler)
		})
	}
}
