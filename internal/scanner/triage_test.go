package scanner

import (
	"testing"
	"time"

	"github.com/arikbautista/meiki/internal/entry"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// date returns a UTC midnight time for the given year, month, day.
func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// makeTriageItem builds an OpenItem with the given priority, due date, and
// capture timestamp. All other fields are set to reasonable defaults.
func makeTriageItem(priority, due string, captureDay time.Time) OpenItem {
	ts := captureDay.Format(time.RFC3339)
	e := entry.Entry{
		ID:        "test-id",
		Timestamp: ts,
		Type:      "todo",
		Content:   "test todo",
		Status:    "open",
		Priority:  priority,
		Due:       due,
	}
	return OpenItem{
		Entry:       e,
		LatestState: e,
		AgeDays:     0,
	}
}

// ---------------------------------------------------------------------------
// Priority: "tomorrow"
// ---------------------------------------------------------------------------

// TestClassifyItem_TomorrowCapturedToday: captured today → not overdue today.
func TestClassifyItem_TomorrowCapturedToday(t *testing.T) {
	today := date(2026, 5, 16)
	item := makeTriageItem("tomorrow", "", today)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal, got %v", triage)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue, got %d", days)
	}
}

// TestClassifyItem_TomorrowCapturedYesterday: captured yesterday → overdue by 1 day today.
func TestClassifyItem_TomorrowCapturedYesterday(t *testing.T) {
	today := date(2026, 5, 16)
	yesterday := date(2026, 5, 15)
	item := makeTriageItem("tomorrow", "", yesterday)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue, got %v", triage)
	}
	if days != 1 {
		t.Errorf("expected 1 day overdue, got %d", days)
	}
}

// TestClassifyItem_TomorrowCapturedTwoDaysAgo: 2 days overdue → TriageOverdue (staleDays=3).
func TestClassifyItem_TomorrowCapturedTwoDaysAgo(t *testing.T) {
	today := date(2026, 5, 16)
	capture := date(2026, 5, 14) // 2 days ago → 2 days overdue
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue, got %v", triage)
	}
	if days != 2 {
		t.Errorf("expected 2 days overdue, got %d", days)
	}
}

// TestClassifyItem_TomorrowStale: 3+ days overdue → TriageStale (staleDays=3).
func TestClassifyItem_TomorrowStale(t *testing.T) {
	today := date(2026, 5, 16)
	capture := date(2026, 5, 13) // 3 days ago → 3 days overdue → stale
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageStale {
		t.Errorf("expected TriageStale, got %v", triage)
	}
	if days != 3 {
		t.Errorf("expected 3 days overdue, got %d", days)
	}
}

// TestClassifyItem_TomorrowCapturedTodayBoundary: captured today → not yet overdue.
// The item was meant for "tomorrow" so it is not overdue until tomorrow.
// days = today - captureDay = 0 → not overdue.
func TestClassifyItem_TomorrowCapturedTodayBoundary(t *testing.T) {
	today := date(2026, 5, 16)
	capture := date(2026, 5, 16) // captured today
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal when captured today, got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue when captured today, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// Priority: "this-week"
// ---------------------------------------------------------------------------

// TestClassifyItem_ThisWeekCapturedMonday: captured Monday 2026-05-11.
// End of ISO week = Sunday 2026-05-17. On 2026-05-11 (Monday): not overdue.
func TestClassifyItem_ThisWeekCapturedMonday_NotOverdue(t *testing.T) {
	// Monday 2026-05-11
	capture := date(2026, 5, 11) // Monday
	today := date(2026, 5, 16)   // Saturday — still same week
	item := makeTriageItem("this-week", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal (within week), got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue, got %d", days)
	}
}

// TestClassifyItem_ThisWeekCapturedMonday_OverdueFollowingMonday:
// captured Monday 2026-05-11; following Monday 2026-05-18 → 1 day overdue.
func TestClassifyItem_ThisWeekCapturedMonday_OverdueFollowingMonday(t *testing.T) {
	capture := date(2026, 5, 11) // Monday; end of week = Sunday 2026-05-17
	today := date(2026, 5, 18)   // following Monday
	item := makeTriageItem("this-week", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue on following Monday, got %v (days=%d)", triage, days)
	}
	if days != 1 {
		t.Errorf("expected 1 day overdue, got %d", days)
	}
}

// TestClassifyItem_ThisWeekEndOfWeekBoundary: captured Monday; today = Sunday end-of-week.
// today == threshold → days = 0 → not overdue.
func TestClassifyItem_ThisWeekEndOfWeekBoundary(t *testing.T) {
	capture := date(2026, 5, 11) // Monday; end of week = Sunday 2026-05-17
	today := date(2026, 5, 17)   // Sunday = end of week
	item := makeTriageItem("this-week", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal on end-of-week day, got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue on end-of-week boundary, got %d", days)
	}
}

// TestClassifyItem_ThisWeekCapturedSunday: captured Sunday 2026-05-10.
// Go's Weekday() treats Sunday as 0 → daysUntilSunday = 0 → end of week = 2026-05-10.
// Monday 2026-05-11 → 1 day overdue.
func TestClassifyItem_ThisWeekCapturedSunday(t *testing.T) {
	capture := date(2026, 5, 10) // Sunday; end of week = 2026-05-10
	today := date(2026, 5, 11)   // Monday → 1 day past end-of-week
	item := makeTriageItem("this-week", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue, got %v (days=%d)", triage, days)
	}
	if days != 1 {
		t.Errorf("expected 1 day overdue, got %d", days)
	}
}

// TestClassifyItem_ThisWeekStale: 3+ days after end of week → TriageStale.
func TestClassifyItem_ThisWeekStale(t *testing.T) {
	capture := date(2026, 5, 11) // Monday; end of week = Sunday 2026-05-17
	today := date(2026, 5, 20)   // Wednesday — 3 days after end of week
	item := makeTriageItem("this-week", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageStale {
		t.Errorf("expected TriageStale, got %v (days=%d)", triage, days)
	}
	if days != 3 {
		t.Errorf("expected 3 days overdue, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// Priority: "someday"
// ---------------------------------------------------------------------------

// TestClassifyItem_Someday: "someday" items are never overdue.
func TestClassifyItem_Someday(t *testing.T) {
	capture := date(2020, 1, 1) // far in the past
	today := date(2026, 5, 16)
	item := makeTriageItem("someday", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal for someday, got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue for someday, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// Due date override
// ---------------------------------------------------------------------------

// TestClassifyItem_DueDate_NotOverdue: due today → not overdue.
func TestClassifyItem_DueDate_NotOverdue(t *testing.T) {
	today := date(2026, 5, 16)
	capture := date(2026, 5, 1)
	item := makeTriageItem("someday", "2026-05-16", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal when due == today, got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue, got %d", days)
	}
}

// TestClassifyItem_DueDate_OneDayOverdue: due 2026-05-10; today 2026-05-11 → 1 day overdue.
func TestClassifyItem_DueDate_OneDayOverdue(t *testing.T) {
	today := date(2026, 5, 11)
	capture := date(2026, 5, 1)
	item := makeTriageItem("someday", "2026-05-10", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue, got %v (days=%d)", triage, days)
	}
	if days != 1 {
		t.Errorf("expected 1 day overdue, got %d", days)
	}
}

// TestClassifyItem_DueDate_Stale: due 2026-05-10; today 2026-05-13 → 3 days overdue → stale.
func TestClassifyItem_DueDate_Stale(t *testing.T) {
	today := date(2026, 5, 13)
	capture := date(2026, 5, 1)
	item := makeTriageItem("someday", "2026-05-10", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageStale {
		t.Errorf("expected TriageStale, got %v (days=%d)", triage, days)
	}
	if days != 3 {
		t.Errorf("expected 3 days overdue, got %d", days)
	}
}

// TestClassifyItem_DueDate_OverridesPriority: due date takes precedence over priority.
// Priority is "tomorrow" (which would be overdue differently), but explicit due wins.
func TestClassifyItem_DueDate_OverridesPriority(t *testing.T) {
	today := date(2026, 5, 16)
	capture := date(2026, 5, 15)
	// Explicit due = 2026-05-10 → 6 days overdue (due takes precedence over priority)
	item := makeTriageItem("tomorrow", "2026-05-10", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageStale {
		t.Errorf("expected TriageStale (due overrides priority), got %v (days=%d)", triage, days)
	}
	if days != 6 {
		t.Errorf("expected 6 days overdue, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// staleDays config
// ---------------------------------------------------------------------------

// TestClassifyItem_CustomStaleDays: with staleDays=5, 4 days overdue → TriageOverdue.
func TestClassifyItem_CustomStaleDays_FourDays(t *testing.T) {
	today := date(2026, 5, 16)
	// "tomorrow" captured 4 days ago → days overdue = 4
	capture := date(2026, 5, 12)
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 5)
	if triage != TriageOverdue {
		t.Errorf("expected TriageOverdue with staleDays=5 and 4 days overdue, got %v (days=%d)", triage, days)
	}
	if days != 4 {
		t.Errorf("expected 4 days overdue, got %d", days)
	}
}

// TestClassifyItem_CustomStaleDays_AtBoundary: with staleDays=5, exactly 5 days overdue → TriageStale.
func TestClassifyItem_CustomStaleDays_AtBoundary(t *testing.T) {
	today := date(2026, 5, 16)
	// "tomorrow" captured 5 days ago → days overdue = 5
	capture := date(2026, 5, 11)
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 5)
	if triage != TriageStale {
		t.Errorf("expected TriageStale with staleDays=5 and 5 days overdue, got %v (days=%d)", triage, days)
	}
	if days != 5 {
		t.Errorf("expected 5 days overdue, got %d", days)
	}
}

// TestClassifyItem_CustomStaleDays_Default: staleDays <= 0 defaults to 3.
func TestClassifyItem_CustomStaleDays_Default(t *testing.T) {
	today := date(2026, 5, 16)
	// captured 3 days ago → 3 days overdue → stale with default staleDays=3
	capture := date(2026, 5, 13)
	item := makeTriageItem("tomorrow", "", capture)

	triage, days := ClassifyItem(item, today, 0) // 0 → default 3
	if triage != TriageStale {
		t.Errorf("expected TriageStale with default staleDays, got %v (days=%d)", triage, days)
	}
	if days != 3 {
		t.Errorf("expected 3 days overdue, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// No priority, no due date
// ---------------------------------------------------------------------------

// TestClassifyItem_NoPriorityNoDue: no priority and no due → never overdue.
func TestClassifyItem_NoPriorityNoDue(t *testing.T) {
	capture := date(2020, 1, 1)
	today := date(2026, 5, 16)
	item := makeTriageItem("", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("expected TriageNormal for no priority/due, got %v (days=%d)", triage, days)
	}
	if days != 0 {
		t.Errorf("expected 0 days overdue, got %d", days)
	}
}

// ---------------------------------------------------------------------------
// Acceptance criteria from spec
// ---------------------------------------------------------------------------

// TestAcceptanceCriteria_TomorrowCapturedYesterdayIs1DayOverdue
func TestAcceptanceCriteria_TomorrowCapturedYesterdayIs1DayOverdue(t *testing.T) {
	today := date(2026, 5, 16)
	yesterday := date(2026, 5, 15)
	item := makeTriageItem("tomorrow", "", yesterday)

	_, days := ClassifyItem(item, today, 3)
	if days != 1 {
		t.Errorf("acceptance: tomorrow captured yesterday should be 1 day overdue, got %d", days)
	}
}

// TestAcceptanceCriteria_TomorrowCapturedTodayNotOverdue
func TestAcceptanceCriteria_TomorrowCapturedTodayNotOverdue(t *testing.T) {
	today := date(2026, 5, 16)
	item := makeTriageItem("tomorrow", "", today)

	triage, _ := ClassifyItem(item, today, 3)
	if triage != TriageNormal {
		t.Errorf("acceptance: tomorrow captured today should not be overdue, got %v", triage)
	}
}

// TestAcceptanceCriteria_ThisWeekCapturedMondayOverdueFollowingMonday
func TestAcceptanceCriteria_ThisWeekCapturedMondayOverdueFollowingMonday(t *testing.T) {
	// Monday 2026-05-11; following Monday = 2026-05-18
	capture := date(2026, 5, 11)
	nextMonday := date(2026, 5, 18)
	item := makeTriageItem("this-week", "", capture)

	triage, _ := ClassifyItem(item, nextMonday, 3)
	if triage == TriageNormal {
		t.Errorf("acceptance: this-week captured Monday should be overdue on following Monday")
	}
}

// TestAcceptanceCriteria_SomedayNeverOverdue
func TestAcceptanceCriteria_SomedayNeverOverdue(t *testing.T) {
	capture := date(2020, 1, 1)
	today := date(2026, 5, 16)
	item := makeTriageItem("someday", "", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage != TriageNormal || days != 0 {
		t.Errorf("acceptance: someday items are never overdue, got triage=%v days=%d", triage, days)
	}
}

// TestAcceptanceCriteria_DueDateOverridesPriority
func TestAcceptanceCriteria_DueDateOverridesPriority(t *testing.T) {
	// due: 2026-05-10, overdue on 2026-05-11 regardless of priority
	today := date(2026, 5, 11)
	capture := date(2026, 5, 1)
	item := makeTriageItem("someday", "2026-05-10", capture)

	triage, days := ClassifyItem(item, today, 3)
	if triage == TriageNormal {
		t.Errorf("acceptance: due 2026-05-10 item should be overdue on 2026-05-11")
	}
	if days != 1 {
		t.Errorf("acceptance: expected 1 day overdue, got %d", days)
	}
}

// TestAcceptanceCriteria_OneTwoDaysOverdueIsTriageOverdue
func TestAcceptanceCriteria_OneTwoDaysOverdueIsTriageOverdue(t *testing.T) {
	for _, daysOld := range []int{1, 2} {
		today := date(2026, 5, 16)
		// "tomorrow" captured daysOld days ago → daysOld days overdue
		capture := today.AddDate(0, 0, -daysOld)
		item := makeTriageItem("tomorrow", "", capture)

		triage, days := ClassifyItem(item, today, 3)
		if triage != TriageOverdue {
			t.Errorf("acceptance: %d days overdue should be TriageOverdue, got %v", daysOld, triage)
		}
		if days != daysOld {
			t.Errorf("acceptance: expected %d days overdue, got %d", daysOld, days)
		}
	}
}

// TestAcceptanceCriteria_ThreePlusDaysOverdueIsTriageStale
func TestAcceptanceCriteria_ThreePlusDaysOverdueIsTriageStale(t *testing.T) {
	for _, daysOld := range []int{3, 5, 10} {
		today := date(2026, 5, 16)
		// "tomorrow" captured daysOld days ago → daysOld days overdue
		capture := today.AddDate(0, 0, -daysOld)
		item := makeTriageItem("tomorrow", "", capture)

		triage, days := ClassifyItem(item, today, 3)
		if triage != TriageStale {
			t.Errorf("acceptance: %d days overdue should be TriageStale with staleDays=3, got %v", daysOld, triage)
		}
		if days != daysOld {
			t.Errorf("acceptance: expected %d days overdue, got %d", daysOld, days)
		}
	}
}
