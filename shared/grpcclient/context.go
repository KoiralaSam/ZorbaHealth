package grpcclient

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type forwardedTokenContextKey struct{}

const forwardedTokenHeader = "x-forwarded-token"

// WithForwardedToken stores the end-user access token both as a context value
// (so in-process handlers can read it) and on the outgoing gRPC metadata (so the
// shared client interceptor forwards it to downstream services). Both
// representations are required: handlers read the context value while the
// transport layer reads the metadata header.
func WithForwardedToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	ctx = context.WithValue(ctx, forwardedTokenContextKey{}, token)
	return metadata.AppendToOutgoingContext(ctx, forwardedTokenHeader, token)
}

func ForwardedTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(forwardedTokenContextKey{}).(string)
	return token, ok && token != ""
}
