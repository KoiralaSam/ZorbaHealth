package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	coreerrors "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

// IPAPIProvider uses an external HTTP API to turn an IP into approximate coordinates.
//
// The API endpoint (including any key/query params) is provided via env:
//   - IP_GEOLOCATION_ENDPOINT_TEMPLATE: must contain a single "%s" placeholder for the IP.
//     Example: "http://ip-api.com/json/%s"
type IPAPIProvider struct {
	endpointTemplate string
	client           *http.Client
}

func NewIPAPIProvider() (outbound.GeolocationProvider, error) {
	endpointTemplate := env.GetString("IP_GEOLOCATION_ENDPOINT_TEMPLATE", "")
	if endpointTemplate == "" {
		return nil, fmt.Errorf("IP_GEOLOCATION_ENDPOINT_TEMPLATE is required")
	}

	timeoutSec := env.GetInt("IP_GEOLOCATION_TIMEOUT_SEC", 5)
	return &IPAPIProvider{
		endpointTemplate: endpointTemplate,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}, nil
}

var _ outbound.GeolocationProvider = (*IPAPIProvider)(nil)

func (p *IPAPIProvider) Geolocate(ctx context.Context, ip string) (*models.Location, error) {
	if ip == "" {
		return nil, fmt.Errorf("%w: empty ip", coreerrors.ErrNoLocationFound)
	}

	ipEscaped := url.PathEscape(ip)
	endpoint := fmt.Sprintf(p.endpointTemplate, ipEscaped)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: ip geolocation http status %d", coreerrors.ErrNoLocationFound, resp.StatusCode)
	}

	var result struct {
		Status string  `json:"status"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		City   string  `json:"city"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("%w: ip geolocation failed for %s (status=%s)", coreerrors.ErrNoLocationFound, ip, result.Status)
	}

	// city-level accuracy per ip geolocation APIs (approx ~5km for many providers)
	return &models.Location{
		Lat:      result.Lat,
		Lng:      result.Lon,
		Method:   "ip-geolocation",
		Accuracy: 5000,
	}, nil
}
