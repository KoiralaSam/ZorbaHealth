package postgres

import (
	"testing"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
)

func TestIsAllowedWelfareRunTransition(t *testing.T) {
	cases := []struct {
		from, to models.WelfareCheckRunStatus
		ok       bool
	}{
		{models.WelfareRunStatusDispatched, models.WelfareRunStatusAnswered, true},
		{models.WelfareRunStatusDispatched, models.WelfareRunStatusCompleted, true},
		{models.WelfareRunStatusDispatched, models.WelfareRunStatusMissed, true},
		{models.WelfareRunStatusDispatched, models.WelfareRunStatusFailed, true},
		{models.WelfareRunStatusAnswered, models.WelfareRunStatusCompleted, true},
		{models.WelfareRunStatusAnswered, models.WelfareRunStatusPending, false},
		{models.WelfareRunStatusCompleted, models.WelfareRunStatusAnswered, false},
		{models.WelfareRunStatusPending, models.WelfareRunStatusAnswered, false},
		{models.WelfareRunStatusClaimed, models.WelfareRunStatusFailed, true},
		{models.WelfareRunStatusDispatched, models.WelfareRunStatusDispatched, true},
	}
	for _, tc := range cases {
		got := isAllowedWelfareRunTransition(tc.from, tc.to)
		if got != tc.ok {
			t.Fatalf("%s -> %s: got %v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}
