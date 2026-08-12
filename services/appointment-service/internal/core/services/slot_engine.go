package services

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/google/uuid"
)

// ComputeSlots expands weekly availability rules over [from, to), subtracts
// exceptions (time-off) and booked appointments, and emits discrete bookable slots.
// Extra-availability exceptions (IsAvailable=true) are merged in as additional windows.
func ComputeSlots(
	staffID, hospitalID uuid.UUID,
	rules []models.AvailabilityRule,
	exceptions []models.AvailabilityException,
	booked []models.Appointment,
	from, to time.Time,
	limit int,
) ([]models.AppointmentSlot, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("to must be after from")
	}
	if limit <= 0 {
		limit = 100
	}

	windows := make([]timeWindow, 0)
	for _, rule := range rules {
		expanded, err := expandRule(rule, from, to)
		if err != nil {
			return nil, err
		}
		windows = append(windows, expanded...)
	}
	for _, ex := range exceptions {
		if ex.IsAvailable && ex.EndsAt.After(from) && ex.StartsAt.Before(to) {
			start := ex.StartsAt
			if start.Before(from) {
				start = from
			}
			end := ex.EndsAt
			if end.After(to) {
				end = to
			}
			dur := int32(30)
			if len(rules) > 0 && rules[0].SlotDurationMinutes > 0 {
				dur = rules[0].SlotDurationMinutes
			}
			tz := "UTC"
			if len(rules) > 0 && rules[0].Timezone != "" {
				tz = rules[0].Timezone
			}
			windows = append(windows, timeWindow{
				start: start, end: end, duration: dur, timezone: tz,
			})
		}
	}

	blocked := make([]timeWindow, 0)
	for _, ex := range exceptions {
		if !ex.IsAvailable && ex.EndsAt.After(from) && ex.StartsAt.Before(to) {
			blocked = append(blocked, timeWindow{start: ex.StartsAt, end: ex.EndsAt})
		}
	}
	for _, appt := range booked {
		if appt.Status != models.AppointmentStatusBooked {
			continue
		}
		blocked = append(blocked, timeWindow{start: appt.StartsAt, end: appt.EndsAt})
	}

	slots := make([]models.AppointmentSlot, 0)
	now := time.Now().UTC()
	for _, w := range windows {
		dur := time.Duration(w.duration) * time.Minute
		if dur <= 0 {
			continue
		}
		cursor := w.start
		for cursor.Add(dur).Before(w.end) || cursor.Add(dur).Equal(w.end) {
			slotEnd := cursor.Add(dur)
			if !slotEnd.After(from) || !cursor.Before(to) {
				cursor = cursor.Add(dur)
				continue
			}
			if !cursor.After(now) {
				cursor = cursor.Add(dur)
				continue
			}
			if overlapsAny(cursor, slotEnd, blocked) {
				cursor = cursor.Add(dur)
				continue
			}
			slots = append(slots, models.AppointmentSlot{
				StartsAt:        cursor.UTC(),
				EndsAt:          slotEnd.UTC(),
				DurationMinutes: w.duration,
				Timezone:        w.timezone,
				StaffID:         staffID,
				HospitalID:      hospitalID,
			})
			if len(slots) >= limit {
				sort.Slice(slots, func(i, j int) bool { return slots[i].StartsAt.Before(slots[j].StartsAt) })
				return slots, nil
			}
			cursor = cursor.Add(dur)
		}
	}

	sort.Slice(slots, func(i, j int) bool { return slots[i].StartsAt.Before(slots[j].StartsAt) })
	return slots, nil
}

type timeWindow struct {
	start    time.Time
	end      time.Time
	duration int32
	timezone string
}

func expandRule(rule models.AvailabilityRule, from, to time.Time) ([]timeWindow, error) {
	loc, err := time.LoadLocation(rule.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", rule.Timezone, err)
	}
	startHM, err := parseHHMM(rule.StartTimeLocal)
	if err != nil {
		return nil, err
	}
	endHM, err := parseHHMM(rule.EndTimeLocal)
	if err != nil {
		return nil, err
	}
	if endHM <= startHM {
		return nil, fmt.Errorf("end_time_local must be after start_time_local")
	}

	effectiveFrom := rule.EffectiveFrom
	if effectiveFrom.IsZero() {
		effectiveFrom = from
	}
	var effectiveUntil *time.Time
	if rule.EffectiveUntil != nil && !rule.EffectiveUntil.IsZero() {
		effectiveUntil = rule.EffectiveUntil
	}

	// Walk calendar days in the staff timezone covering [from, to).
	dayStart := time.Date(from.In(loc).Year(), from.In(loc).Month(), from.In(loc).Day(), 0, 0, 0, 0, loc)
	endDay := time.Date(to.In(loc).Year(), to.In(loc).Month(), to.In(loc).Day(), 0, 0, 0, 0, loc)
	if to.In(loc).After(endDay) {
		// include the day of `to` if to has a time component past midnight
	}

	out := make([]timeWindow, 0)
	for d := dayStart; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		if int(d.Weekday()) != rule.Weekday {
			continue
		}
		dayDate := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		effFromDate := time.Date(effectiveFrom.Year(), effectiveFrom.Month(), effectiveFrom.Day(), 0, 0, 0, 0, time.UTC)
		if dayDate.Before(effFromDate) {
			continue
		}
		if effectiveUntil != nil {
			effUntilDate := time.Date(effectiveUntil.Year(), effectiveUntil.Month(), effectiveUntil.Day(), 0, 0, 0, 0, time.UTC)
			if dayDate.After(effUntilDate) {
				continue
			}
		}
		winStart := time.Date(d.Year(), d.Month(), d.Day(), startHM/60, startHM%60, 0, 0, loc)
		winEnd := time.Date(d.Year(), d.Month(), d.Day(), endHM/60, endHM%60, 0, 0, loc)
		if winEnd.Before(from) || !winStart.Before(to) {
			continue
		}
		out = append(out, timeWindow{
			start:    winStart,
			end:      winEnd,
			duration: rule.SlotDurationMinutes,
			timezone: rule.Timezone,
		})
	}
	return out, nil
}

func parseHHMM(s string) (int, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time %q; want HH:MM", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour in %q", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid minute in %q", s)
	}
	return h*60 + m, nil
}

func overlapsAny(start, end time.Time, blocked []timeWindow) bool {
	for _, b := range blocked {
		if start.Before(b.end) && end.After(b.start) {
			return true
		}
	}
	return false
}
