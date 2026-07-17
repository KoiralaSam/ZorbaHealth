package main

import (
	"testing"
	"time"

	patientportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patientportal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWelfareCheckFromProtoMapsReasonCode(t *testing.T) {
	now := time.Now().UTC()
	record := welfareCheckFromProto(&patientportalpb.WelfareCheck{
		Id:                     "check-1",
		PatientId:              "patient-1",
		ScheduledAt:            timestamppb.New(now),
		Timezone:               "America/Chicago",
		ReasonCode:             "daily_checkup",
		ReasonDetail:           "note",
		Status:                 "scheduled",
		LatestRunId:            "run-1",
		LatestRunStatus:        "pending",
		LatestRunAttempts:      1,
		LatestRunFailureReason: "",
	})
	if record.ReasonCode != "daily_checkup" {
		t.Fatalf("reason_code=%s", record.ReasonCode)
	}
	if record.LatestRunStatus != "pending" {
		t.Fatalf("latest_run_status=%s", record.LatestRunStatus)
	}
	if record.Timezone != "America/Chicago" {
		t.Fatalf("timezone=%s", record.Timezone)
	}
}
