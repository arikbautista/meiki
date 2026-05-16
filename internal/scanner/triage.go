package scanner

import (
	"time"
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
func daysOverdue(item OpenItem, today time.Time) int {
	orig := item.Entry

	// Normalise today to midnight UTC.
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	// Parse capture timestamp.
	captureTime, err := time.Parse(time.RFC3339, orig.Timestamp)
	if err != nil {
		return 0
	}
	captureDay := time.Date(captureTime.Year(), captureTime.Month(), captureTime.Day(), 0, 0, 0, 0, time.UTC)

	// Explicit due date takes precedence over priority-based calculation.
	if orig.Due != "" {
		dueDay, err := time.Parse("2006-01-02", orig.Due)
		if err != nil {
			return 0
		}
		// Overdue when today > due date, i.e. today is at least dueDay+1.
		days := int(today.Sub(dueDay).Hours() / 24)
		if days <= 0 {
			return 0
		}
		return days
	}

	// Priority-based calculation.
	switch orig.Priority {
	case "tomorrow":
		// "tomorrow" means the item was meant to be done the day after it was
		// captured. It becomes overdue starting the day after capture — i.e., the
		// very next day the item was supposed to be completed.
		//
		// Acceptance criterion: captured yesterday → 1 day overdue today.
		// If captureDay = today-1, days = today - captureDay = 1. Correct.
		// If captureDay = today, days = 0 → not overdue. Correct.
		days := int(today.Sub(captureDay).Hours() / 24)
		if days <= 0 {
			return 0
		}
		return days

	case "this-week":
		// End of the ISO week that the capture day belongs to.
		// ISO week starts on Monday; end of week is Sunday.
		// We find the Sunday that ends the capture week.
		weekday := int(captureDay.Weekday()) // 0=Sunday, 1=Mon, ..., 6=Sat
		// Days until end of week (Sunday). If captureDay is Monday (1),
		// end is 6 days later. If Sunday (0), end is same day.
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
func ClassifyItem(item OpenItem, today time.Time, staleDays int) (ItemTriage, int) {
	if staleDays <= 0 {
		staleDays = 3
	}

	overdue := daysOverdue(item, today)
	if overdue <= 0 {
		return TriageNormal, 0
	}
	if overdue >= staleDays {
		return TriageStale, overdue
	}
	return TriageOverdue, overdue
}
