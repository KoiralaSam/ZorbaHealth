package auth

import (
	"context"
	"errors"
	"os"
	"strings"

	grpc "google.golang.org/grpc"
	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	InternalTokenHeader   = "x-internal-token"
	InternalServiceHeader = "x-internal-service"
	ForwardedTokenHeader  = "x-forwarded-token"
)

type ContextKey string

const internalServiceContextKey ContextKey = "internal_service_name"

type InternalServerConfig struct {
	SharedSecret     string
	RequireServiceID bool
	AllowedServices  map[string]struct{}
}

func ServiceNameFromContext(ctx context.Context) string {
	value, _ := ctx.Value(internalServiceContextKey).(string)
	return value
}

func UnaryServerInterceptor(cfg InternalServerConfig) grpcserver.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		_ *grpcserver.UnaryServerInfo,
		handler grpcserver.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		expected := strings.TrimSpace(cfg.SharedSecret)
		if expected == "" {
			expected = strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_SECRET"))
		}
		values := md.Get(InternalTokenHeader)
		if len(values) == 0 || expected == "" || values[0] != expected {
			return nil, status.Error(codes.Unauthenticated, "invalid internal token")
		}

		service := strings.TrimSpace(first(md.Get(InternalServiceHeader)))
		if cfg.RequireServiceID && service == "" {
			return nil, status.Error(codes.Unauthenticated, "missing internal service identity")
		}
		if len(cfg.AllowedServices) > 0 && service != "" {
			if _, ok := cfg.AllowedServices[service]; !ok {
				return nil, status.Error(codes.PermissionDenied, "internal service not allowed")
			}
		}

		ctx = context.WithValue(ctx, internalServiceContextKey, service)
		return handler(ctx, req)
	}
}

func UnaryClientInterceptor(serviceName, sharedSecret string, requireForwarded bool) grpc.UnaryClientInterceptor {
	serviceName = strings.TrimSpace(serviceName)
	sharedSecret = strings.TrimSpace(sharedSecret)
	if sharedSecret == "" {
		sharedSecret = strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_SECRET"))
	}
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if sharedSecret == "" {
			return errors.New("INTERNAL_SERVICE_SECRET is not set")
		}
		pairs := []string{InternalTokenHeader, sharedSecret}
		if serviceName != "" {
			pairs = append(pairs, InternalServiceHeader, serviceName)
		}
		if requireForwarded {
			alreadyOutgoing := false
			forwarded := strings.TrimSpace(first(metadata.ValueFromIncomingContext(ctx, ForwardedTokenHeader)))
			if forwarded == "" {
				if md, ok := metadata.FromOutgoingContext(ctx); ok {
					forwarded = strings.TrimSpace(first(md.Get(ForwardedTokenHeader)))
					alreadyOutgoing = forwarded != ""
				}
			}
			if forwarded == "" {
				return errors.New("forwarded token missing from context")
			}
			if !alreadyOutgoing {
				pairs = append(pairs, ForwardedTokenHeader, forwarded)
			}
		}
		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func TLSCredentials(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	if strings.TrimSpace(certFile) == "" || strings.TrimSpace(keyFile) == "" || strings.TrimSpace(caFile) == "" {
		return insecure.NewCredentials(), nil
	}
	return credentials.NewClientTLSFromFile(caFile, "")
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
