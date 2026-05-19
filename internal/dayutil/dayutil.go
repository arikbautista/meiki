package dayutil

import "time"

// LogicalDay returns the logical calendar date for t in the given location.
// If the local hour of t is before dayStartHour, the previous calendar day
// is returned. The returned time is midnight (00:00) of the logical day
// in the given location.
func LogicalDay(t time.Time, loc *time.Location, dayStartHour int) time.Time {
	local := t.In(loc)
	y, m, d := local.Date()
	if local.Hour() < dayStartHour {
		prev := time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, -1)
		return prev
	}
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// LogicalDayStr returns LogicalDay formatted as "2006-01-02".
func LogicalDayStr(t time.Time, loc *time.Location, dayStartHour int) string {
	return LogicalDay(t, loc, dayStartHour).Format("2006-01-02")
}
