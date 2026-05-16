// Package brief provides briefing output and debouncing.
package brief

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
)

// BriefItem is a single item included in the briefing.
type BriefItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Project     string `json:"project,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AgeDays     int    `json:"age_days"`
	OverdueDays int    `json:"overdue_days"`
	Triage      string `json:"triage"` // "normal", "overdue", "stale"
}

// Briefing holds all the data for a morning briefing.
type Briefing struct {
	ReviewSummary string      // condensed review from most recent review file
	OpenTodos     []BriefItem // todos sorted by priority then capture date, capped at limit
	OpenBlockers  []BriefItem // all open blockers
	NeedsTriage   []BriefItem // items 3+ days overdue (stale)
	Welcome       bool        // true on fresh install with no history
	Debounced     bool        // true if brief was suppressed by debounce logic
}

// priorityOrder maps priority string to sort key (lower = higher priority).
var priorityOrder = map[string]int{
	"tomorrow":  0,
	"this-week": 1,
	"someday":   2,
}

// triageName converts scanner.ItemTriage to a string for JSON output.
func triageName(t scanner.ItemTriage) string {
	switch t {
	case scanner.TriageOverdue:
		return "overdue"
	case scanner.TriageStale:
		return "stale"
	default:
		return "normal"
	}
}

// GenerateBriefing produces a Briefing given the data directory, config, and state.
// It handles debouncing, fresh-install detection, and content generation.
func GenerateBriefing(dataDir string, cfg config.Config, state config.State) (*Briefing, error) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// --- Debounce check ---
	if state.LastBriefTS != "" {
		lastBrief, err := time.Parse(time.RFC3339, state.LastBriefTS)
		if err == nil {
			lastBriefDay := time.Date(lastBrief.Year(), lastBrief.Month(), lastBrief.Day(), 0, 0, 0, 0, time.UTC)
			if lastBriefDay.Equal(today) {
				// Brief already produced today. Check for new entries since last_brief_ts.
				hasNew, err := hasNewEntriesSince(dataDir, now, lastBrief)
				if err != nil {
					return nil, fmt.Errorf("check new entries: %w", err)
				}
				if !hasNew {
					return &Briefing{Debounced: true}, nil
				}
			}
		}
	}

	// --- Scan open items ---
	todos, blockers, err := scanner.ScanOpenItems(dataDir, cfg.UI.OpenScanDays)
	if err != nil {
		return nil, fmt.Errorf("scan open items: %w", err)
	}

	// --- Review summary ---
	reviewSummary, err := findRecentReviewSummary(dataDir, today)
	if err != nil {
		// Non-fatal: proceed without a review summary.
		reviewSummary = ""
	}

	// --- Fresh-install detection ---
	if len(todos) == 0 && len(blockers) == 0 && reviewSummary == "" {
		return &Briefing{Welcome: true}, nil
	}

	// --- Classify and split items ---
	staleDays := cfg.UI.StaleTriageDays

	var mainTodos []BriefItem
	var triageTodos []BriefItem

	for _, item := range todos {
		triage, overdueDays := scanner.ClassifyItem(item, today, staleDays)
		bi := BriefItem{
			ID:          item.Entry.ID,
			Type:        "todo",
			Content:     item.Entry.Content,
			Project:     item.Entry.Project,
			Priority:    item.Entry.Priority,
			AgeDays:     item.AgeDays,
			OverdueDays: overdueDays,
			Triage:      triageName(triage),
		}
		if triage == scanner.TriageStale {
			triageTodos = append(triageTodos, bi)
		} else {
			mainTodos = append(mainTodos, bi)
		}
	}

	// Sort main todos by priority then capture timestamp.
	sort.SliceStable(mainTodos, func(i, j int) bool {
		pi := priorityOrder[mainTodos[i].Priority]
		pj := priorityOrder[mainTodos[j].Priority]
		if pi != pj {
			return pi < pj
		}
		return mainTodos[i].ID < mainTodos[j].ID // ULID order = capture order
	})

	// Apply the brief_max_open_todos limit.
	limit := cfg.UI.BriefMaxOpenTodos
	if limit > 0 && len(mainTodos) > limit {
		mainTodos = mainTodos[:limit]
	}

	// Build blockers BriefItems.
	var briefBlockers []BriefItem
	for _, item := range blockers {
		triage, overdueDays := scanner.ClassifyItem(item, today, staleDays)
		briefBlockers = append(briefBlockers, BriefItem{
			ID:          item.Entry.ID,
			Type:        "blocker",
			Content:     item.Entry.Content,
			Project:     item.Entry.Project,
			AgeDays:     item.AgeDays,
			OverdueDays: overdueDays,
			Triage:      triageName(triage),
		})
	}

	// Stale items: combine stale todos and stale blockers.
	var needsTriage []BriefItem
	needsTriage = append(needsTriage, triageTodos...)
	for _, item := range blockers {
		triage, overdueDays := scanner.ClassifyItem(item, today, staleDays)
		if triage == scanner.TriageStale {
			needsTriage = append(needsTriage, BriefItem{
				ID:          item.Entry.ID,
				Type:        "blocker",
				Content:     item.Entry.Content,
				Project:     item.Entry.Project,
				AgeDays:     item.AgeDays,
				OverdueDays: overdueDays,
				Triage:      "stale",
			})
			// Remove from briefBlockers to avoid duplication.
		}
	}
	// Filter stale blockers out of briefBlockers.
	filtered := briefBlockers[:0]
	for _, bi := range briefBlockers {
		if bi.Triage != "stale" {
			filtered = append(filtered, bi)
		}
	}
	briefBlockers = filtered

	return &Briefing{
		ReviewSummary: reviewSummary,
		OpenTodos:     mainTodos,
		OpenBlockers:  briefBlockers,
		NeedsTriage:   needsTriage,
	}, nil
}

// hasNewEntriesSince returns true if today's JSONL file contains any entry
// with a timestamp strictly after lastBrief.
func hasNewEntriesSince(dataDir string, now time.Time, lastBrief time.Time) (bool, error) {
	y := now.Format("2006")
	m := now.Format("01")
	d := now.Format("2006-01-02")
	path := filepath.Join(dataDir, "entries", y, m, d+".jsonl")

	entries, err := entry.ReadEntriesFromPath(path)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		ts, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		if ts.After(lastBrief) {
			return true, nil
		}
	}
	return false, nil
}

// findRecentReviewSummary looks back up to 7 days from yesterday and returns
// a condensed version of the most recent review markdown found. Returns empty
// string if no review is found (not an error).
func findRecentReviewSummary(dataDir string, today time.Time) (string, error) {
	for i := 1; i <= 7; i++ {
		date := today.AddDate(0, 0, -i)
		y := date.Format("2006")
		m := date.Format("01")
		d := date.Format("2006-01-02")
		path := filepath.Join(dataDir, "reviews", y, m, d+".md")

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		return condenseReview(string(data)), nil
	}
	return "", nil
}

// condenseReview extracts section headings and the first few bullet points
// from each section of a review markdown. This keeps the summary brief.
func condenseReview(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	inSection := false
	bulletCount := 0
	const maxBulletsPerSection = 3

	for _, line := range lines {
		// Top-level heading: include as-is.
		if strings.HasPrefix(line, "# ") {
			out = append(out, line)
			inSection = false
			bulletCount = 0
			continue
		}
		// Section heading (##): start a new section.
		if strings.HasPrefix(line, "## ") {
			out = append(out, line)
			inSection = true
			bulletCount = 0
			continue
		}
		// Bullet points within a section: include up to maxBulletsPerSection.
		if inSection && strings.HasPrefix(line, "- ") {
			if bulletCount < maxBulletsPerSection {
				out = append(out, line)
				bulletCount++
			} else if bulletCount == maxBulletsPerSection {
				out = append(out, "  ...")
				bulletCount++
			}
		}
	}

	return strings.Join(out, "\n")
}

// FormatHuman renders a Briefing as human-readable text.
// Returns empty string if Debounced is true.
func FormatHuman(b *Briefing) string {
	if b.Debounced {
		return ""
	}
	if b.Welcome {
		return "Welcome to meiki. Entries you capture during AI sessions will appear in your next briefing."
	}

	var sb strings.Builder

	if b.ReviewSummary != "" {
		sb.WriteString("## Yesterday's Review\n\n")
		sb.WriteString(b.ReviewSummary)
		sb.WriteString("\n\n")
	}

	if len(b.OpenTodos) > 0 {
		fmt.Fprintf(&sb, "## Open Todos (%d)\n\n", len(b.OpenTodos))
		for _, item := range b.OpenTodos {
			writeBriefItem(&sb, item)
		}
		sb.WriteString("\n")
	}

	if len(b.OpenBlockers) > 0 {
		fmt.Fprintf(&sb, "## Open Blockers (%d)\n\n", len(b.OpenBlockers))
		for _, item := range b.OpenBlockers {
			writeBriefItem(&sb, item)
		}
		sb.WriteString("\n")
	}

	if len(b.NeedsTriage) > 0 {
		fmt.Fprintf(&sb, "## Needs Triage (%d)\n\n", len(b.NeedsTriage))
		sb.WriteString("The following items are 3+ days overdue and need your attention:\n\n")
		for _, item := range b.NeedsTriage {
			writeBriefItem(&sb, item)
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// writeBriefItem appends a single formatted item line to sb.
func writeBriefItem(sb *strings.Builder, item BriefItem) {
	priority := item.Priority
	if priority == "" {
		if item.Type == "blocker" {
			priority = "blocker"
		} else {
			priority = "someday"
		}
	}

	project := item.Project
	if item.OverdueDays > 0 {
		if project != "" {
			fmt.Fprintf(sb, "- [%s] %s (%s) — %d %s overdue\n",
				priority, item.Content, project, item.OverdueDays, plural("day", item.OverdueDays))
		} else {
			fmt.Fprintf(sb, "- [%s] %s — %d %s overdue\n",
				priority, item.Content, item.OverdueDays, plural("day", item.OverdueDays))
		}
	} else {
		if project != "" {
			fmt.Fprintf(sb, "- [%s] %s (%s)\n", priority, item.Content, project)
		} else {
			fmt.Fprintf(sb, "- [%s] %s\n", priority, item.Content)
		}
	}
}

// plural returns word pluralised by appending "s" if count != 1.
func plural(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
