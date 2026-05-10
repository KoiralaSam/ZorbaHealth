package mappers

import (
	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
)

func SummaryToProto(s *models.HospitalSummary) *pb.HospitalSummaryResponse {
	if s == nil {
		return &pb.HospitalSummaryResponse{}
	}

	return &pb.HospitalSummaryResponse{
		TotalConsentedPatients: int32(s.TotalConsentedPatients),
		TotalCalls_30D:         int32(s.TotalCalls30d),
		EmergencyEvents_30D:    int32(s.EmergencyEvents30d),
		AvgCallDurationSeconds: s.AvgCallDurationSeconds,
		RecordsIndexed:         int32(s.RecordsIndexed),
		ActivePatients_7D:      int32(s.ActivePatients7d),
	}
}

func CallVolumeToProto(points []models.CallVolumePoint) *pb.CallVolumeResponse {
	pbPoints := make([]*pb.CallVolumePoint, 0, len(points))
	for _, p := range points {
		pbPoints = append(pbPoints, &pb.CallVolumePoint{
			Date:           p.Date.Format("2006-01-02"),
			TotalCalls:     int32(p.TotalCalls),
			CompletedCalls: int32(p.CompletedCalls),
			EmergencyCalls: int32(p.EmergencyCalls),
			AvgDurationSec: p.AvgDurationSec,
		})
	}

	return &pb.CallVolumeResponse{Points: pbPoints}
}

func ConditionsToProto(conditions []models.ConditionCount) *pb.TopConditionsResponse {
	pbConditions := make([]*pb.ConditionCount, 0, len(conditions))
	for _, c := range conditions {
		pbConditions = append(pbConditions, &pb.ConditionCount{
			ConditionName: c.ConditionName,
			PatientCount:  int32(c.PatientCount),
			Percentage:    c.Percentage,
		})
	}

	return &pb.TopConditionsResponse{Conditions: pbConditions}
}

func ToolUsageToProto(stats []models.ToolUsageStat) *pb.ToolUsageResponse {
	pbStats := make([]*pb.ToolUsageStat, 0, len(stats))
	for _, s := range stats {
		pbStats = append(pbStats, &pb.ToolUsageStat{
			Tool:         s.Tool,
			SuccessCount: int32(s.SuccessCount),
			ErrorCount:   int32(s.ErrorCount),
			DeniedCount:  int32(s.DeniedCount),
			SuccessRate:  s.SuccessRate,
		})
	}

	return &pb.ToolUsageResponse{Stats: pbStats}
}

func ActivityToProto(events []models.ActivityEvent) *pb.RecentActivityResponse {
	pbEvents := make([]*pb.ActivityEvent, 0, len(events))
	for _, e := range events {
		pbEvents = append(pbEvents, &pb.ActivityEvent{
			Timestamp: e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			Tool:      e.Tool,
			ActorType: e.ActorType,
			Outcome:   e.Outcome,
			SessionId: e.SessionID,
		})
	}

	return &pb.RecentActivityResponse{Events: pbEvents}
}

func PlatformSummaryToProto(s *models.PlatformSummary) *pb.PlatformSummaryResponse {
	if s == nil {
		return &pb.PlatformSummaryResponse{}
	}

	return &pb.PlatformSummaryResponse{
		TotalHospitals:       int32(s.TotalHospitals),
		TotalPatients:        int32(s.TotalPatients),
		TotalCalls_30D:       int32(s.TotalCalls30d),
		TotalEmergencies_30D: int32(s.TotalEmergencies30d),
		AvgCallDurationSec:   s.AvgCallDurationSec,
		ActiveHospitals_7D:   int32(s.ActiveHospitals7d),
	}
}

func HospitalBreakdownToProto(stats []models.HospitalStat) *pb.HospitalBreakdownResponse {
	pbStats := make([]*pb.HospitalStat, 0, len(stats))
	for _, s := range stats {
		pbStats = append(pbStats, &pb.HospitalStat{
			HospitalId:     s.HospitalID,
			HospitalName:   s.HospitalName,
			PatientCount:   int32(s.PatientCount),
			CallCount:      int32(s.CallCount),
			EmergencyCount: int32(s.EmergencyCount),
		})
	}

	return &pb.HospitalBreakdownResponse{Hospitals: pbStats}
}

func PatientCallHistoryToProto(history *models.PatientCallHistory) *pb.PatientCallHistoryResponse {
	if history == nil {
		return &pb.PatientCallHistoryResponse{}
	}

	pbCalls := make([]*pb.PatientCall, 0, len(history.Calls))
	for _, c := range history.Calls {
		pbCalls = append(pbCalls, &pb.PatientCall{
			StartedAt:    c.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
			Duration:     c.Duration,
			Summary:      c.Summary,
			HadEmergency: c.HadEmergency,
		})
	}

	return &pb.PatientCallHistoryResponse{
		TotalCalls: int32(history.TotalCalls),
		Calls:      pbCalls,
	}
}
