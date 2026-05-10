package mappers

import (
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
)

func accuracyToProto(accuracy float64) string {
	// Domain accuracy is in metres.
	// GPS fixes are typically <= 1km; IP geolocation is ~city-level (thousands of metres).
	if accuracy <= 1000 {
		return "precise"
	}
	return "city-level"
}

func methodToProto(method string) string {
	// Proto expects "gps" | "ip-approximate".
	if method == "gps" {
		return "gps"
	}
	return "ip-approximate"
}

// LocationToProto maps domain Location models to location-service gRPC proto types.
func LocationToProto(loc *models.Location) *pb.GetLocationResponse {
	if loc == nil {
		return &pb.GetLocationResponse{}
	}

	return &pb.GetLocationResponse{
		Lat:      loc.Lat,
		Lng:      loc.Lng,
		Method:   methodToProto(loc.Method),
		Accuracy: accuracyToProto(loc.Accuracy),
	}
}

// HospitalToProto maps domain Hospital models to location-service gRPC proto types.
//
// Note: the current domain `Hospital` model is intentionally minimal; proto has additional fields
// (directions_url, phone) that are filled with empty strings until the provider output is defined.
func HospitalToProto(h *models.Hospital) *pb.FindHospitalResponse {
	if h == nil {
		return &pb.FindHospitalResponse{}
	}

	return &pb.FindHospitalResponse{
		Name:          h.Name,
		Address:       h.Address,
		DirectionsUrl: "",
		Phone:         "",
	}
}
