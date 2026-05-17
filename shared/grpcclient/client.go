package grpcclient

import (
	"context"
	"errors"
	"os"

	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func Dial(addr string) (*grpc.ClientConn, error) {
	return DialInsecure(addr, grpc.WithUnaryInterceptor(injectAuthMetadata))
}

func DialInsecure(addr string, extraOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	dialOptions = append(dialOptions, extraOptions...)
	return grpc.NewClient(addr, dialOptions...)
}

func injectAuthMetadata(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	internal := os.Getenv("INTERNAL_SERVICE_SECRET")
	if internal == "" {
		return errors.New("INTERNAL_SERVICE_SECRET is not set")
	}

	forwarded, ok := ForwardedTokenFromContext(ctx)
	if !ok {
		return errors.New("forwarded token missing from context")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-internal-token", internal,
		"x-forwarded-token", forwarded,
	)

	return invoker(ctx, method, req, reply, cc, opts...)
}
