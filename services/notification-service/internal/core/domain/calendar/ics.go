package calendar

import (
	"fmt"
	"strings"
	"time"
)

// BuildMeetingRequestICS returns RFC5545 calendar content for a scheduled visit.
func BuildMeetingRequestICS(uid, summary, description, organizerEmail, attendeeEmail string, start time.Time, durationMinutes int, timezone string) string {
	end := start.Add(time.Duration(durationMinutes) * time.Minute)
	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}
	startLocal := start.In(loc)
	endLocal := end.In(loc)
	format := "20060102T150405"
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Zorba Health//Scheduling//EN",
		"METHOD:REQUEST",
		"BEGIN:VEVENT",
		"UID:" + escapeICS(uid),
		"DTSTAMP:" + time.Now().UTC().Format(format) + "Z",
		"DTSTART;TZID=" + escapeICS(timezone) + ":" + startLocal.Format(format),
		"DTEND;TZID=" + escapeICS(timezone) + ":" + endLocal.Format(format),
		"SUMMARY:" + escapeICS(summary),
		"DESCRIPTION:" + escapeICS(description),
		"ORGANIZER;CN=Zorba Health:MAILTO:" + organizerEmail,
		"ATTENDEE;CUTYPE=INDIVIDUAL;ROLE=REQ-PARTICIPANT;PARTSTAT=NEEDS-ACTION;RSVP=TRUE:MAILTO:" + attendeeEmail,
		"STATUS:CONFIRMED",
		"SEQUENCE:0",
		"END:VEVENT",
		"END:VCALENDAR",
	}
	return strings.Join(lines, "\r\n")
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	return s
}

func MeetingUID(meetingID, recipientEmail string) string {
	return fmt.Sprintf("zorba-meeting-%s-%s@zorba.health", meetingID, strings.ToLower(recipientEmail))
}
