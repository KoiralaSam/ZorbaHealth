package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

var _ outbound.HospitalFinder = (*NominatimHospitalFinder)(nil)

type NominatimHospitalFinder struct {
	baseURL  string
	client   *http.Client
	fallback []models.Hospital
}

func NewNominatimHospitalFinder() outbound.HospitalFinder {
	timeoutSec := env.GetInt("HOSPITAL_LOOKUP_TIMEOUT_SEC", 5)
	baseURL := env.GetString("HOSPITAL_LOOKUP_BASE_URL", "https://nominatim.openstreetmap.org/search")
	return &NominatimHospitalFinder{
		baseURL: strings.TrimSpace(baseURL),
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		fallback: []models.Hospital{
			{
				Name:      "Zorba General Hospital",
				Address:   "100 Demo Health Way, Boston, MA",
				Lat:       42.3601,
				Lng:       -71.0589,
				PlaceType: "hospital",
			},
			{
				Name:      "Mercy Urgent Care Center",
				Address:   "250 Example Avenue, Cambridge, MA",
				Lat:       42.3736,
				Lng:       -71.1097,
				PlaceType: "urgent_care",
			},
			{
				Name:      "City Pharmacy Hub",
				Address:   "500 Sample Street, Somerville, MA",
				Lat:       42.3876,
				Lng:       -71.0995,
				PlaceType: "pharmacy",
			},
		},
	}
}

func (n *NominatimHospitalFinder) FindNearest(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error) {
	if placeType == "" {
		placeType = "hospital"
	}

	result, err := n.fetchNearest(ctx, lat, lng, placeType)
	if err == nil && result != nil {
		return result, nil
	}

	return n.closestFallback(lat, lng, placeType), nil
}

func (n *NominatimHospitalFinder) fetchNearest(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error) {
	queryURL, err := url.Parse(n.baseURL)
	if err != nil {
		return nil, err
	}

	values := queryURL.Query()
	values.Set("format", "jsonv2")
	values.Set("limit", "5")
	values.Set("q", placeType)
	values.Set("lat", fmt.Sprintf("%f", lat))
	values.Set("lon", fmt.Sprintf("%f", lng))
	queryURL.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zorba-health-location-service/1.0")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hospital lookup failed with status %d", resp.StatusCode)
	}

	var payload []struct {
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		Type        string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var best *models.Hospital
	bestDistance := math.MaxFloat64
	for _, item := range payload {
		itemLat, parseErr := parseFloat(item.Lat)
		if parseErr != nil {
			continue
		}
		itemLng, parseErr := parseFloat(item.Lon)
		if parseErr != nil {
			continue
		}
		distance := haversineKm(lat, lng, itemLat, itemLng)
		if distance < bestDistance {
			bestDistance = distance
			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = firstSegment(item.DisplayName)
			}
			best = &models.Hospital{
				Name:      name,
				Address:   item.DisplayName,
				Lat:       itemLat,
				Lng:       itemLng,
				PlaceType: placeType,
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no hospitals found")
	}
	return best, nil
}

func (n *NominatimHospitalFinder) closestFallback(lat, lng float64, placeType string) *models.Hospital {
	best := n.fallback[0]
	bestDistance := math.MaxFloat64
	for _, candidate := range n.fallback {
		if candidate.PlaceType != placeType && placeType != "hospital" {
			continue
		}
		distance := haversineKm(lat, lng, candidate.Lat, candidate.Lng)
		if distance < bestDistance {
			bestDistance = distance
			best = candidate
		}
	}
	return &best
}

func parseFloat(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

func firstSegment(value string) string {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return value
	}
	return strings.TrimSpace(parts[0])
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
