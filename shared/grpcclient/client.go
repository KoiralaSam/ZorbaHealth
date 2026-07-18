package grpcclient

import (
	"os"

	sharedgrpcauth "github.com/KoiralaSam/ZorbaHealth/shared/grpc/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(addr string) (*grpc.ClientConn, error) {
	serviceName := os.Getenv("GRPC_CLIENT_SERVICE_NAME")
	if serviceName == "" {
		serviceName = os.Getenv("SERVICE_NAME")
	}
	return DialInsecure(addr, grpc.WithUnaryInterceptor(sharedgrpcauth.UnaryClientInterceptor(serviceName, os.Getenv("INTERNAL_SERVICE_SECRET"), true)))
}

func DialInsecure(addr string, extraOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	dialOptions = append(dialOptions, extraOptions...)
	return grpc.NewClient(addr, dialOptions...)
}

// injectAuthMetadata is now handled by shared/grpc/auth.UnaryClientInterceptor.
