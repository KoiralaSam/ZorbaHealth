package mappers

import (
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	appointmentpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/appointment"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func AppointmentToProto(a *models.Appointment) *appointmentpb.Appointment {
	if a == nil {
		return nil
	}
	return &appointmentpb.Appointment{
		Id:                a.ID.String(),
		PatientId:         a.PatientID.String(),
		StaffId:           a.StaffID.String(),
		HospitalId:        a.HospitalID.String(),
		StartsAt:          timestamppb.New(a.StartsAt),
		DurationMinutes:   a.DurationMinutes,
		Timezone:          a.Timezone,
		Type:              string(a.Type),
		Status:            string(a.Status),
		Channel:           string(a.Channel),
		Title:             a.Title,
		Notes:             a.Notes,
		CorrelationId:     a.CorrelationID.String(),
		VoiceSessionId:    a.VoiceSessionID,
		BookedByActorType: a.BookedByActorType,
		BookedByActorId:   a.BookedByActorID,
		JoinUrl:           a.JoinURL,
		LivekitRoomName:   a.LiveKitRoomName,
		PatientToken:      a.PatientToken,
		StaffToken:        a.StaffToken,
		CreatedAt:         timestamppb.New(a.CreatedAt),
		UpdatedAt:         timestamppb.New(a.UpdatedAt),
	}
}

func SlotToProto(s models.AppointmentSlot) *appointmentpb.AppointmentSlot {
	return &appointmentpb.AppointmentSlot{
		StartsAt:        timestamppb.New(s.StartsAt),
		EndsAt:          timestamppb.New(s.EndsAt),
		DurationMinutes: s.DurationMinutes,
		Timezone:        s.Timezone,
		StaffId:         s.StaffID.String(),
		HospitalId:      s.HospitalID.String(),
	}
}

func RuleToProto(r models.AvailabilityRule) *appointmentpb.AvailabilityRule {
	out := &appointmentpb.AvailabilityRule{
		Id:                  r.ID.String(),
		StaffId:             r.StaffID.String(),
		HospitalId:          r.HospitalID.String(),
		Weekday:             int32(r.Weekday),
		StartTimeLocal:      r.StartTimeLocal,
		EndTimeLocal:        r.EndTimeLocal,
		SlotDurationMinutes: r.SlotDurationMinutes,
		Timezone:            r.Timezone,
		EffectiveFrom:       timestamppb.New(r.EffectiveFrom),
	}
	if r.EffectiveUntil != nil {
		out.EffectiveUntil = timestamppb.New(*r.EffectiveUntil)
	}
	return out
}

func RuleFromProto(r *appointmentpb.AvailabilityRule, staffID, hospitalID uuid.UUID) models.AvailabilityRule {
	rule := models.AvailabilityRule{
		StaffID:             staffID,
		HospitalID:          hospitalID,
		Weekday:             int(r.GetWeekday()),
		StartTimeLocal:      r.GetStartTimeLocal(),
		EndTimeLocal:        r.GetEndTimeLocal(),
		SlotDurationMinutes: r.GetSlotDurationMinutes(),
		Timezone:            r.GetTimezone(),
	}
	if r.GetEffectiveFrom() != nil {
		rule.EffectiveFrom = r.GetEffectiveFrom().AsTime()
	} else {
		rule.EffectiveFrom = time.Now().UTC()
	}
	if r.GetEffectiveUntil() != nil {
		t := r.GetEffectiveUntil().AsTime()
		rule.EffectiveUntil = &t
	}
	return rule
}

func ExceptionToProto(e models.AvailabilityException) *appointmentpb.AvailabilityException {
	return &appointmentpb.AvailabilityException{
		Id:          e.ID.String(),
		StaffId:     e.StaffID.String(),
		HospitalId:  e.HospitalID.String(),
		StartsAt:    timestamppb.New(e.StartsAt),
		EndsAt:      timestamppb.New(e.EndsAt),
		Reason:      e.Reason,
		IsAvailable: e.IsAvailable,
	}
}
