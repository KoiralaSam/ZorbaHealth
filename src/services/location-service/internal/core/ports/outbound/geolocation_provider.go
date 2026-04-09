package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
)

// GeolocationProvider is the secondary port for IP-based fallback geolocation.
type GeolocationProvider interface {
	Geolocate(ctx context.Context, ip string) (*models.Location, error)
}

// HospitalFinder is the secondary port for finding nearby facilities.
type HospitalFinder interface {
	FindNearest(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error)
}
