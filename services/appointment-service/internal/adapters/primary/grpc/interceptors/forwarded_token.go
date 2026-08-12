package interceptors

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ForwardedTokenInterceptor(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if tokens := md.Get("x-forwarded-token"); len(tokens) > 0 && tokens[0] != "" {
			ctx = grpcclient.WithForwardedToken(ctx, tokens[0])
		}
	}
	return handler(ctx, req)
}
