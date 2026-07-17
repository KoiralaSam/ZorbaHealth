package models

import (
	"time"

	"github.com/google/uuid"
)

type WelfareCheckReason string

const (
	WelfareReasonMedicationReminder WelfareCheckReason = "medication_reminder"
	WelfareReasonMentalWellbeing    WelfareCheckReason = "mental_wellbeing"
	WelfareReasonDailyCheckup       WelfareCheckReason = "daily_checkup"
	WelfareReasonSymptomFollowUp    WelfareCheckReason = "symptom_follow_up"
	WelfareReasonCarePlanReminder   WelfareCheckReason = "care_plan_reminder"
	WelfareReasonOther              WelfareCheckReason = "other"
)

type WelfareCheckStatus string

const (
	WelfareCheckStatusScheduled WelfareCheckStatus = "scheduled"
	WelfareCheckStatusCancelled WelfareCheckStatus = "cancelled"
	WelfareCheckStatusCompleted WelfareCheckStatus = "completed"
	WelfareCheckStatusMissed    WelfareCheckStatus = "missed"
	WelfareCheckStatusFailed    WelfareCheckStatus = "failed"
)

type WelfareCheckRunStatus string

const (
	WelfareRunStatusPending    WelfareCheckRunStatus = "pending"
	WelfareRunStatusClaimed    WelfareCheckRunStatus = "claimed"
	WelfareRunStatusDispatched WelfareCheckRunStatus = "dispatched"
	WelfareRunStatusAnswered   WelfareCheckRunStatus = "answered"
	WelfareRunStatusMissed     WelfareCheckRunStatus = "missed"
	WelfareRunStatusCompleted  WelfareCheckRunStatus = "completed"
	WelfareRunStatusFailed     WelfareCheckRunStatus = "failed"
	WelfareRunStatusCancelled  WelfareCheckRunStatus = "cancelled"
)

type CreateWelfareCheckCommand struct {
	PatientID    uuid.UUID
	ScheduledAt  time.Time
	Timezone     string
	ReasonCode   WelfareCheckReason
	ReasonDetail string
	ActorID      string
}

type WelfareCheck struct {
	ID                     uuid.UUID
	PatientID              uuid.UUID
	ScheduledAt            time.Time
	Timezone               string
	ReasonCode             WelfareCheckReason
	ReasonDetail           string
	Status                 WelfareCheckStatus
	RecurrenceRule         string
	RecurrenceStartsAt     *time.Time
	RecurrenceEndsAt       *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	CancelledAt            *time.Time
	LatestRunID            string
	LatestRunStatus        string
	LatestRunAttempts      int32
	LatestRunFailureReason string
}

type WelfareCheckRun struct {
	ID                  uuid.UUID
	RequestID           uuid.UUID
	PatientID           uuid.UUID
	ScheduledAt         time.Time
	Status              WelfareCheckRunStatus
	Attempts            int32
	LastAttemptAt       *time.Time
	NextAttemptAt       *time.Time
	LiveKitRoomName     string
	LiveKitRoomSID      string
	LiveKitDispatchID   string
	LiveKitSIPCallID    string
	FailureReason       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	RequestReasonCode   WelfareCheckReason
	RequestReasonDetail string
	RequestTimezone     string
	PatientPhoneNumber  string
	PatientFullName     string
	PatientUserID       uuid.UUID
}

type ListWelfareChecksFilter struct {
	PatientID        uuid.UUID
	IncludeCancelled bool
	Limit            int32
}

type WelfareCheckDispatchResult struct {
	RunID               uuid.UUID
	RequestID           uuid.UUID
	RoomName            string
	RoomSID             string
	DispatchID          string
	SIPCallID           string
	ParticipantID       string
	ParticipantIdentity string
}

type UpdateWelfareRunLifecycleCommand struct {
	PatientID string
	RunID     string
	Status    WelfareCheckRunStatus
	Reason    string
	ActorID   string
}
