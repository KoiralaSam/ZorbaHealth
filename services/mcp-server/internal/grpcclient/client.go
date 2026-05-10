package grpcclient

import "google.golang.org/grpc"

import sharedgrpcclient "github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"

func Dial(addr string) (*grpc.ClientConn, error) {
	return sharedgrpcclient.Dial(addr)
}
