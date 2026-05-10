package grpcclient

import (
	"context"
	"errors"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(injectAuthMetadata),
	)
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
