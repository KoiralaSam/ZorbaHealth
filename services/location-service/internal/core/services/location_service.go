package services

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
)

type LocationService struct {
	repo       outbound.LocationRepository
	registry   outbound.ConnectionRegistry
	geolocator outbound.GeolocationProvider
	hospitals  outbound.HospitalFinder

	pendingMu           sync.Mutex
	pendingVoiceSession map[string]string // patientID -> LiveKit/voice sessionID awaiting WS connect
}

var _ inbound.LocationService = (*LocationService)(nil)

// patientLiveConn adapts inbound.PatientLiveChannel to outbound.Connection.
type patientLiveConn struct {
	ch       inbound.PatientLiveChannel
	clientIP string
}

func (p *patientLiveConn) WriteJSON(v any) error { return p.ch.WriteJSON(v) }
func (p *patientLiveConn) Close() error          { return p.ch.Close() }
func (p *patientLiveConn) ClientIP() string      { return p.clientIP }

func NewLocationService(
	repo outbound.LocationRepository,
	registry outbound.ConnectionRegistry,
	geolocator outbound.GeolocationProvider,
	hospitals outbound.HospitalFinder,
) inbound.LocationService {
	return &LocationService{
		repo:                repo,
		registry:            registry,
		geolocator:          geolocator,
		hospitals:           hospitals,
		pendingVoiceSession: make(map[string]string),
	}
}

func (s *LocationService) StoreLocation(ctx context.Context, sessionID string, loc models.Location) error {
	if loc.Lat == 0 && loc.Lng == 0 {
		return errors.ErrInvalidCoordinates
	}
	loc.SessionID = sessionID
	return s.repo.Save(ctx, sessionID, loc)
}

func (s *LocationService) GetLocation(ctx context.Context, sessionID string) (*models.Location, error) {
	// 1. Try Redis — live GPS from the mobile app
	loc, err := s.repo.Get(ctx, sessionID)
	if err == nil {
		return loc, nil
	}

	// 2. No GPS in Redis — return not found and let the caller
	//    decide whether to fall back to IP geolocation
	return nil, fmt.Errorf("%w: session %s", errors.ErrNoLocationFound, sessionID)
}

func (s *LocationService) GetLocationWithFallback(ctx context.Context, sessionID, callerIP string) (*models.Location, error) {
	loc, err := s.GetLocation(ctx, sessionID)
	if err == nil {
		return loc, nil
	}
	// Fall back to IP geolocation
	return s.geolocator.Geolocate(ctx, callerIP)
}

func (s *LocationService) FindNearestHospital(ctx context.Context, lat, lng float64, placeType string) (*models.Hospital, error) {
	if placeType == "" {
		placeType = "hospital"
	}
	return s.hospitals.FindNearest(ctx, lat, lng, placeType)
}

func (s *LocationService) RegisterPatientLiveChannel(ctx context.Context, patientID string, ch inbound.PatientLiveChannel) {
	s.registry.Register(patientID, &patientLiveConn{ch: ch, clientIP: ch.ClientIP()})
	log.Printf("patient %s connected to location WS", patientID)

	if sessionID, ok := s.takePendingVoiceSession(patientID); ok {
		cmd := models.LocationCommand{Command: "start_location", SessionID: sessionID}
		if err := s.registry.Send(patientID, cmd); err != nil {
			log.Printf("deliver pending start_location failed patient=%s: %v", patientID, err)
		} else {
			log.Printf("delivered pending start_location session=%s patient=%s", sessionID, patientID)
		}
	}
}

func (s *LocationService) UnregisterPatientLiveChannel(_ context.Context, patientID string) {
	s.registry.Unregister(patientID)
}

// HandleCallEvent is called by the RabbitMQ consumer when call lifecycle events arrive.
// It pushes a WebSocket command to the patient's connected app.
func (s *LocationService) HandleCallEvent(ctx context.Context, event events.CallEvent) error {
	var cmd models.LocationCommand

	switch event.EventType {
	case "call.started":
		cmd = models.LocationCommand{
			Command:   "start_location",
			SessionID: event.SessionID,
		}
	case "call.ended":
		cmd = models.LocationCommand{
			Command:   "stop_location",
			SessionID: event.SessionID,
		}
		s.clearPendingVoiceSession(event.PatientID)
		// Clean up Redis entry when call ends
		_ = s.repo.Delete(ctx, event.SessionID)
	default:
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}

	if err := s.registry.Send(event.PatientID, cmd); err != nil {
		if event.EventType == "call.started" {
			s.setPendingVoiceSession(event.PatientID, event.SessionID)
			log.Printf("patient %s not on WS yet; queued start_location for session %s", event.PatientID, event.SessionID)
		} else {
			log.Printf("no WS connection for patient %s: %v", event.PatientID, err)
		}
	} else if event.EventType == "call.started" {
		log.Printf("sent start_location to patient %s session=%s (browser should prompt for GPS)", event.PatientID, event.SessionID)
	}

	return nil
}

func (s *LocationService) setPendingVoiceSession(patientID, sessionID string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.pendingVoiceSession[patientID] = sessionID
}

func (s *LocationService) takePendingVoiceSession(patientID string) (string, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	sessionID, ok := s.pendingVoiceSession[patientID]
	if ok {
		delete(s.pendingVoiceSession, patientID)
	}
	return sessionID, ok
}

func (s *LocationService) clearPendingVoiceSession(patientID string) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	delete(s.pendingVoiceSession, patientID)
}
