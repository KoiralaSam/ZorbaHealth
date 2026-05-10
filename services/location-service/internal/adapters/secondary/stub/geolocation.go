package stub

import (
	"context"
	"fmt"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
)

var _ outbound.GeolocationProvider = (*NoopGeolocation)(nil)

// NoopGeolocation is a placeholder until a real IP geolocation provider is wired.
type NoopGeolocation struct{}

func NewNoopGeolocation() *NoopGeolocation { return &NoopGeolocation{} }

func (n *NoopGeolocation) Geolocate(ctx context.Context, ip string) (*models.Location, error) {
	_ = ctx
	_ = ip
	return nil, fmt.Errorf("%w: ip geolocation not configured", errors.ErrNoLocationFound)
}
