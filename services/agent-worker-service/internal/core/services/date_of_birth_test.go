package services

import (
	"testing"
	"time"
)

func TestParseDateOfBirth(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{
			name: "ISO",
			in:   "2005-01-02",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "USSlash",
			in:   "01/02/2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "STTSpacesAroundDashes",
			in:   "01 - 02 - 2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "STTSpacesNoDashes",
			in:   "01 02 2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "STTMMDDDashYYYY",
			in:   "0102-2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "DigitsOnlyMMDDYYYY",
			in:   "01022005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "AmbiguousDashAssumeUS",
			in:   "01-02-2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "UnambiguousDDMM",
			in:   "13-02-2005",
			want: time.Date(2005, time.February, 13, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "MonthName",
			in:   "January 2, 2005",
			want: time.Date(2005, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "Empty",
			in:      "   ",
			wantErr: true,
		},
		{
			name:    "Invalid",
			in:      "2005-13-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateOfBirth(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Year() != tt.want.Year() || got.Month() != tt.want.Month() || got.Day() != tt.want.Day() {
				t.Fatalf("got=%04d-%02d-%02d want=%04d-%02d-%02d", got.Year(), got.Month(), got.Day(), tt.want.Year(), tt.want.Month(), tt.want.Day())
			}
		})
	}
}
