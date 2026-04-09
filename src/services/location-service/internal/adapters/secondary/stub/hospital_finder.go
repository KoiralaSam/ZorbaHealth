package stub

import (
	"context"
	"errors"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
)

var _ outbound.HospitalFinder = (*NoopHospitalFinder)(nil)

// NoopHospitalFinder is a placeholder until Places / hospital lookup is wired.
type NoopHospitalFinder struct{}

func NewNoopHospitalFinder() *NoopHospitalFinder { return &NoopHospitalFinder{} }

func (n *NoopHospitalFinder) FindNearest(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error) {
	_ = ctx
	_ = lat
	_ = lng
	_ = placeType
	return nil, errors.New("hospital lookup not configured")
}
