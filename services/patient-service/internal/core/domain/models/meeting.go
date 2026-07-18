package models

import (
	"time"

	"github.com/google/uuid"
)

type MeetingChannel string

const (
	MeetingChannelVoice     MeetingChannel = "voice"
	MeetingChannelPortal    MeetingChannel = "portal"
	MeetingChannelDashboard MeetingChannel = "dashboard"
)

type MeetingStatus string

const (
	MeetingStatusPending   MeetingStatus = "pending"
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusCancelled MeetingStatus = "cancelled"
	MeetingStatusFailed    MeetingStatus = "failed"
)

type ScheduleMeetingCommand struct {
	PatientID       uuid.UUID
	StaffID         uuid.UUID
	HospitalID      uuid.UUID
	StartsAt        time.Time
	DurationMinutes int32
	Timezone        string
	Title           string
	Notes           string
	Channel         MeetingChannel
	CorrelationID   uuid.UUID
	VoiceSessionID  string
	SendSMS         bool
	ActorType       string
	ActorID         string
}

type ScheduledMeeting struct {
	ID                 uuid.UUID
	PatientID          uuid.UUID
	StaffID            uuid.UUID
	HospitalID         uuid.UUID
	CreatedByActorType string
	CreatedByActorID   string
	StartsAt           time.Time
	DurationMinutes    int32
	Timezone           string
	Title              string
	Notes              string
	JoinURL            string
	Status             MeetingStatus
	CorrelationID      uuid.UUID
	VoiceSessionID     string
	SendSMS            bool
	Channel            MeetingChannel
	CreatedAt          time.Time
	LiveKitRoomName    string
	LiveKitRoomSID     string
	PatientToken       string
	StaffToken         string
}

type StaffSummary struct {
	StaffID    uuid.UUID
	HospitalID uuid.UUID
	Name       string
	Role       string
	Email      string
}

type ListMeetingsFilter struct {
	PatientID        *uuid.UUID
	StaffID          *uuid.UUID
	HospitalID       *uuid.UUID
	IncludeCancelled bool
	Limit            int32
}

type ScheduleActor struct {
	ActorType  string
	ActorID    string
	PatientID  string
	StaffID    string
	HospitalID string
}
