package interceptors

import (
	"context"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	sharedgrpcauth "github.com/KoiralaSam/ZorbaHealth/shared/grpc/auth"
	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

func Chain() grpcserver.ServerOption {
	return grpcserver.ChainUnaryInterceptor(
		InternalAuthInterceptor,
		optionalClaimsInterceptor,
	)
}

func optionalClaimsInterceptor(
	ctx context.Context,
	req any,
	info *grpcserver.UnaryServerInfo,
	handler grpcserver.UnaryHandler,
) (any, error) {
	if info.FullMethod == "/audit.AuditService/AppendAuditEvent" {
		return handler(ctx, req)
	}
	return ClaimsInterceptor(ctx, req, info, handler)
}
