package models

import (
	"time"
)

type Location struct {
	Lat        float64
	Lng        float64
	Accuracy   float64 // metres
	Method     string  // "gps" | "ip-geolocation" | "demo-hardcoded"
	SessionID  string
	CapturedAt time.Time
}

type LocationCommand struct {
	Command   string // "start_location" | "stop_location"
	SessionID string
}

// Hospital represents a nearby facility returned by the location provider.
// Fields are intentionally minimal for now; extend when the provider output is defined.
type Hospital struct {
	Name      string
	Address   string
	Lat       float64
	Lng       float64
	PlaceType string
}
