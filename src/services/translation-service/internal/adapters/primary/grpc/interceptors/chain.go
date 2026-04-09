package interceptors

import grpcserver "google.golang.org/grpc"

func Chain() grpcserver.ServerOption {
	return grpcserver.ChainUnaryInterceptor(
		InternalAuthInterceptor,
		ClaimsInterceptor,
	)
}
