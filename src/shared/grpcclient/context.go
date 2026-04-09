package grpcclient

import "context"

type forwardedTokenContextKey struct{}

func WithForwardedToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, forwardedTokenContextKey{}, token)
}

func ForwardedTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(forwardedTokenContextKey{}).(string)
	return token, ok && token != ""
}
