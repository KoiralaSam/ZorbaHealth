package grpc

import (
	"context"
	"errors"

	grpcmappers "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/grpc/mappers"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LocationGRPCHandler is the primary gRPC adapter for location-service.
// It maps protobuf requests/responses to the inbound core service.
type LocationGRPCHandler struct {
	pb.UnimplementedLocationServiceServer
	Service inbound.LocationService
}

func NewLocationGRPCHandler(server *grpc.Server, svc inbound.LocationService) *LocationGRPCHandler {
	h := &LocationGRPCHandler{Service: svc}
	pb.RegisterLocationServiceServer(server, h)
	return h
}

func (h *LocationGRPCHandler) GetLocation(ctx context.Context, req *pb.GetLocationRequest) (*pb.GetLocationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	loc, err := h.Service.GetLocation(ctx, req.SessionId)
	if err != nil {
		switch {
		case errors.Is(err, domainerrors.ErrNoLocationFound):
			return nil, status.Error(codes.NotFound, "location not found")
		case errors.Is(err, domainerrors.ErrInvalidCoordinates):
			return nil, status.Error(codes.InvalidArgument, "invalid coordinates")
		default:
			return nil, status.Error(codes.Internal, "get location failed: "+err.Error())
		}
	}

	return grpcmappers.LocationToProto(loc), nil
}

func (h *LocationGRPCHandler) FindNearestHospital(ctx context.Context, req *pb.FindHospitalRequest) (*pb.FindHospitalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	hospital, err := h.Service.FindNearestHospital(ctx, req.Lat, req.Lng, req.PlaceType)
	if err != nil {
		return nil, status.Error(codes.Internal, "hospital lookup failed: "+err.Error())
	}

	return grpcmappers.HospitalToProto(hospital), nil
}
