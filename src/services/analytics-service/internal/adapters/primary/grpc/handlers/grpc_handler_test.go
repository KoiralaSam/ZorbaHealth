package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
)

type stubAnalyticsService struct {
	hospitalSummary    *models.HospitalSummary
	callVolume         []models.CallVolumePoint
	topConditions      []models.ConditionCount
	toolUsage          []models.ToolUsageStat
	recentActivity     []models.ActivityEvent
	platformSummary    *models.PlatformSummary
	hospitalBreakdown  []models.HospitalStat
	patientCallHistory *models.PatientCallHistory
	err                error
}

func (s *stubAnalyticsService) GetHospitalSummary(context.Context, string) (*models.HospitalSummary, error) {
	return s.hospitalSummary, s.err
}

func (s *stubAnalyticsService) GetCallVolume(context.Context, string, string, string) ([]models.CallVolumePoint, error) {
	return s.callVolume, s.err
}

func (s *stubAnalyticsService) GetTopConditions(context.Context, string, int) ([]models.ConditionCount, error) {
	return s.topConditions, s.err
}

func (s *stubAnalyticsService) GetToolUsage(context.Context, string, string) ([]models.ToolUsageStat, error) {
	return s.toolUsage, s.err
}

func (s *stubAnalyticsService) GetRecentActivity(context.Context, string, int) ([]models.ActivityEvent, error) {
	return s.recentActivity, s.err
}

func (s *stubAnalyticsService) GetPlatformSummary(context.Context) (*models.PlatformSummary, error) {
	return s.platformSummary, s.err
}

func (s *stubAnalyticsService) GetHospitalBreakdown(context.Context, int, string) ([]models.HospitalStat, error) {
	return s.hospitalBreakdown, s.err
}

func (s *stubAnalyticsService) GetPatientCallHistory(context.Context, string, int) (*models.PatientCallHistory, error) {
	return s.patientCallHistory, s.err
}

func TestNewAnalyticsGRPCHandlerRegisters(t *testing.T) {
	server := grpc.NewServer()
	handler := NewAnalyticsGRPCHandler(server, &stubAnalyticsService{}, nil)
	if handler == nil {
		t.Fatal("expected handler")
	}
}

func TestGetCallVolumeMapsResponse(t *testing.T) {
	handler := &AnalyticsGRPCHandler{
		Service: &stubAnalyticsService{
			callVolume: []models.CallVolumePoint{
				{
					Date:           time.Date(2026, 4, 5, 12, 30, 0, 0, time.FixedZone("CDT", -5*60*60)),
					TotalCalls:     12,
					CompletedCalls: 10,
					EmergencyCalls: 2,
					AvgDurationSec: 45.5,
				},
			},
		},
	}

	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType:  sharedauth.ActorStaff,
		HospitalID: "hospital-1",
		StaffID:    "staff-1",
	})

	resp, err := handler.GetCallVolume(ctx, &pb.CallVolumeRequest{
		HospitalId:  "hospital-1",
		Period:      "30d",
		Granularity: "day",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(resp.Points))
	}
	if resp.Points[0].Date != "2026-04-05" {
		t.Fatalf("unexpected date mapping: %s", resp.Points[0].Date)
	}
}

func TestGetHospitalSummaryNilRequest(t *testing.T) {
	handler := &AnalyticsGRPCHandler{Service: &stubAnalyticsService{}}

	_, err := handler.GetHospitalSummary(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestGetHospitalSummaryRejectsPatientActor(t *testing.T) {
	handler := &AnalyticsGRPCHandler{Service: &stubAnalyticsService{}}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorPatient,
		PatientID: "patient-1",
	})

	_, err := handler.GetHospitalSummary(ctx, &pb.HospitalSummaryRequest{HospitalId: "hospital-1"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestGetPatientCallHistoryRejectsPatientMismatch(t *testing.T) {
	handler := &AnalyticsGRPCHandler{Service: &stubAnalyticsService{}}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorPatient,
		PatientID: "patient-1",
	})

	_, err := handler.GetPatientCallHistory(ctx, &pb.PatientCallHistoryRequest{PatientId: "patient-2"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestGetPlatformSummaryAllowsAdmin(t *testing.T) {
	handler := &AnalyticsGRPCHandler{
		Service: &stubAnalyticsService{
			platformSummary: &models.PlatformSummary{TotalHospitals: 3},
		},
	}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorAdmin,
		AdminID:   "admin-1",
	})

	resp, err := handler.GetPlatformSummary(ctx, &pb.PlatformSummaryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalHospitals != 3 {
		t.Fatalf("unexpected summary: %+v", resp)
	}
}
