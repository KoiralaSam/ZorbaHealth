package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
)

// LocationService is the primary port — all adapters drive the core through this.
type LocationService interface {
	// Called by HTTP handler when app pushes GPS coordinates
	StoreLocation(ctx context.Context, sessionID string, loc models.Location) error

	// Called by gRPC handler when mcp-server needs the patient's location
	GetLocation(ctx context.Context, sessionID string) (*models.Location, error)

	// Called by gRPC handler for hospital lookup
	FindNearestHospital(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error)

	// Called by RabbitMQ consumer when a call lifecycle event arrives
	HandleCallEvent(ctx context.Context, event events.CallEvent) error

	// RegisterPatientLiveChannel records the active realtime channel for a patient (WebSocket connect).
	RegisterPatientLiveChannel(ctx context.Context, patientID string, ch PatientLiveChannel)

	// UnregisterPatientLiveChannel clears the patient's channel (WebSocket disconnect).
	UnregisterPatientLiveChannel(ctx context.Context, patientID string)
}
