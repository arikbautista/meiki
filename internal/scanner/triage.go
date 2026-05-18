package scanner

import (
	"time"

	"github.com/arikbautista/meiki/internal/dayutil"
)

// ItemTriage classifies how overdue an open item is.
type ItemTriage int

const (
	// TriageNormal means the item is not overdue.
	TriageNormal ItemTriage = iota
	// TriageOverdue means the item is 1 to (staleDays-1) days overdue.
	TriageOverdue
	// TriageStale means the item is staleDays or more days overdue.
	TriageStale
)

// daysOverdue returns how many days past the overdue threshold the item is,
// given today's date. Returns 0 if the item is not overdue.
//
// Overdue rules (explicit due takes precedence):
//   - due is set:              overdue when today > due date
//   - priority "tomorrow":    overdue when today > captureDay + 1
//   - priority "this-week":   overdue when today > end of ISO week of capture
//   - priority "someday":     never overdue
//   - no priority, no due:    never overdue
func daysOverdue(item OpenItem, today time.Time, loc *time.Location, dayStartHour int) int {
	orig := item.Entry

	// Parse capture timestamp.
	captureTime, err := time.Parse(time.RFC3339, orig.Timestamp)
	if err != nil {
		return 0
	}
	captureDay := dayutil.LogicalDay(captureTime, loc, dayStartHour)

	// today is already a logical day passed by the caller.

	// Explicit due date takes precedence over priority-based calculation.
	if orig.Due != "" {
		dueDay, err := time.ParseInLocation("2006-01-02", orig.Due, loc)
		if err != nil {
			return 0
		}
		days := int(today.Sub(dueDay).Hours() / 24)
		if days <= 0 {
			return 0
		}
		return days
	}

	// Priority-based calculation.
	switch orig.Priority {
	case "tomorrow":
		days := int(today.Sub(captureDay).Hours() / 24)
		if days <= 0 {
			return 0
		}
		return days

	case "this-week":
		weekday := int(captureDay.Weekday())
		var daysUntilSunday int
		if weekday == 0 {
			daysUntilSunday = 0
		} else {
			daysUntilSunday = 7 - weekday
		}
		endOfWeek := captureDay.AddDate(0, 0, daysUntilSunday)
		days := int(today.Sub(endOfWeek).Hours() / 24)
		if days <= 0 {
			return 0
		}
		return days

	case "someday":
		return 0

	default:
		// No priority and no due date — never overdue.
		return 0
	}
}

// ClassifyItem returns the triage classification and the number of days overdue
// for the given open item, relative to today.
//
// staleDays is the threshold at which an item transitions from TriageOverdue to
// TriageStale. The config default is 3.
func ClassifyItem(item OpenItem, today time.Time, staleDays int, loc *time.Location, dayStartHour int) (ItemTriage, int) {
	if staleDays <= 0 {
		staleDays = 3
	}

	overdue := daysOverdue(item, today, loc, dayStartHour)
	if overdue <= 0 {
		return TriageNormal, 0
	}
	if overdue >= staleDays {
		return TriageStale, overdue
	}
	return TriageOverdue, overdue
}
