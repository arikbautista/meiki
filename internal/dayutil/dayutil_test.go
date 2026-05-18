package dayutil

import (
	"testing"
	"time"
)

func TestLogicalDay(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	tests := []struct {
		name         string
		t            time.Time
		loc          *time.Location
		dayStartHour int
		wantDate     string
	}{
		{
			name:         "2am local rolls back to previous day",
			t:            time.Date(2026, 5, 18, 2, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-17",
		},
		{
			name:         "4:59am rolls back",
			t:            time.Date(2026, 5, 18, 4, 59, 59, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-17",
		},
		{
			name:         "exactly 5am stays on current day",
			t:            time.Date(2026, 5, 18, 5, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-18",
		},
		{
			name:         "noon stays on current day",
			t:            time.Date(2026, 5, 18, 12, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-18",
		},
		{
			name:         "11pm stays on current day",
			t:            time.Date(2026, 5, 18, 23, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-18",
		},
		{
			name:         "midnight with dayStartHour=0 stays on current day",
			t:            time.Date(2026, 5, 18, 0, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 0,
			wantDate:     "2026-05-18",
		},
		{
			name:         "3am with dayStartHour=0 stays on current day",
			t:            time.Date(2026, 5, 18, 3, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 0,
			wantDate:     "2026-05-18",
		},
		{
			name:         "dayStartHour=23 rolls back at 10pm",
			t:            time.Date(2026, 5, 18, 22, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 23,
			wantDate:     "2026-05-17",
		},
		{
			name:         "dayStartHour=23 stays at 11pm",
			t:            time.Date(2026, 5, 18, 23, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 23,
			wantDate:     "2026-05-18",
		},
		{
			name:         "UTC input converted to local timezone",
			t:            time.Date(2026, 5, 18, 6, 0, 0, 0, time.UTC), // 2am EDT
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-17",
		},
		{
			name:         "midnight rolls back with dayStartHour=5",
			t:            time.Date(2026, 5, 18, 0, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2026-05-17",
		},
		{
			name:         "Jan 1 2am rolls back to previous year",
			t:            time.Date(2026, 1, 1, 2, 0, 0, 0, ny),
			loc:          ny,
			dayStartHour: 5,
			wantDate:     "2025-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LogicalDay(tt.t, tt.loc, tt.dayStartHour)
			gotStr := got.Format("2006-01-02")
			if gotStr != tt.wantDate {
				t.Errorf("LogicalDay() = %s, want %s", gotStr, tt.wantDate)
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
				t.Errorf("LogicalDay() time = %v, want midnight", got)
			}
			if got.Location() != tt.loc {
				t.Errorf("LogicalDay() location = %v, want %v", got.Location(), tt.loc)
			}
		})
	}
}

func TestLogicalDayStr(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	got := LogicalDayStr(time.Date(2026, 5, 18, 2, 0, 0, 0, ny), ny, 5)
	if got != "2026-05-17" {
		t.Errorf("LogicalDayStr() = %q, want %q", got, "2026-05-17")
	}
}
