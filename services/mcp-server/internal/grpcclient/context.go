package grpcclient

import (
	"context"

	sharedgrpcclient "github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
)

func WithForwardedToken(ctx context.Context, token string) context.Context {
	return sharedgrpcclient.WithForwardedToken(ctx, token)
}

func ForwardedTokenFromContext(ctx context.Context) (string, bool) {
	return sharedgrpcclient.ForwardedTokenFromContext(ctx)
}
