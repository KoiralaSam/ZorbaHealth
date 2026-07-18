package interceptors

import (
	"context"
	"log"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
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

	forwardedToken := sharedauth.NormalizeBearerToken(tokens[0])
	claims, err := sharedauth.VerifyToken(forwardedToken)
	if err != nil {
		log.Printf("forwarded token verification failed: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid forwarded token")
	}

	ctx = sharedauth.WithClaims(ctx, claims)
	ctx = grpcclient.WithForwardedToken(ctx, forwardedToken)
	return handler(ctx, req)
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
			if !requiresForwardedClaims(info.FullMethod) {
				return handler(ctx, req)
			}
			return ClaimsInterceptor(ctx, req, info, handler)
		})
	}
}

func requiresForwardedClaims(fullMethod string) bool {
	switch fullMethod {
	case healthpb.HealthRecordService_IngestDocument_FullMethodName,
		healthpb.HealthRecordService_IngestFHIRBundle_FullMethodName:
		return false
	default:
		return true
	}
}
