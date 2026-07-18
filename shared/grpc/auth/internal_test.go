package auth

import (
	"context"
	"testing"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptorAcceptsAllowedService(t *testing.T) {
	interceptor := UnaryServerInterceptor(InternalServerConfig{
		SharedSecret:     "topsecret",
		RequireServiceID: true,
		AllowedServices: map[string]struct{}{
			"mcp-server": {},
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		InternalTokenHeader, "topsecret",
		InternalServiceHeader, "mcp-server",
	))

	_, err := interceptor(ctx, struct{}{}, &grpcserver.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		if got := ServiceNameFromContext(ctx); got != "mcp-server" {
			t.Fatalf("expected service name in context, got %q", got)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnaryServerInterceptorRejectsMissingIdentity(t *testing.T) {
	interceptor := UnaryServerInterceptor(InternalServerConfig{
		SharedSecret:     "topsecret",
		RequireServiceID: true,
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		InternalTokenHeader, "topsecret",
	))
	_, err := interceptor(ctx, struct{}{}, &grpcserver.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}
