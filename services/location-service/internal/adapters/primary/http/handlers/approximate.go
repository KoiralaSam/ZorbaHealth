package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/secondary/geolocation"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
)

// ApproximateLocationHandler returns approximate coordinates from the HTTP client's IP.
type ApproximateLocationHandler struct {
	Geolocator outbound.GeolocationProvider
	Auth       AuthValidator
}

func (h *ApproximateLocationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	setCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = extractBearerToken(r)
	}
	if _, err := h.Auth.ExtractPatientID(token); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ip := clientIPFromRequest(r)
	if geolocation.IsUngeolocatableIP(ip) {
		http.Error(w, "approximate location unavailable for local connections", http.StatusServiceUnavailable)
		return
	}

	loc, err := h.Geolocator.Geolocate(r.Context(), ip)
	if err != nil {
		log.Printf("approximate location failed ip=%s: %v", ip, err)
		http.Error(w, "approximate location unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"lat":      loc.Lat,
		"lng":      loc.Lng,
		"accuracy": loc.Accuracy,
		"method":   "ip-geolocation",
	})
}

func setCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
}
