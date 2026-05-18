// Package review provides daily review markdown generation.
package review

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
)

// GenerateReview reads today's JSONL entries and open items, and produces a
// structured markdown review string. It uses dataDir for file I/O (so tests can
// redirect to a temp directory) and cfg for scanner configuration.
func GenerateReview(dataDir string, date time.Time, cfg config.Config) (string, error) {
	// Read today's entries from dataDir.
	entryPath := entryFilePath(dataDir, date)
	entries, err := entry.ReadEntriesFromPath(entryPath)
	if err != nil {
		return "", fmt.Errorf("read entries: %w", err)
	}

	// Scan open items from dataDir.
	loc := cfg.Location()
	todos, blockers, err := scanner.ScanOpenItems(dataDir, cfg.UI.OpenScanDays, date, loc, cfg.UI.DayStartHour)
	if err != nil {
		return "", fmt.Errorf("scan open items: %w", err)
	}

	return buildMarkdown(date, entries, todos, blockers, cfg), nil
}

// entryFilePath mirrors the logic in entry.EntryFilePath but accepts an
// explicit dataDir so tests can redirect I/O to a temp directory.
func entryFilePath(dataDir string, date time.Time) string {
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	return filepath.Join(dataDir, "entries", y, m, d+".jsonl")
}

// ReviewFilePath returns the path to the review markdown file for the given date.
// Reviews are stored at <dataDir>/reviews/YYYY/MM/YYYY-MM-DD.md.
func ReviewFilePath(dataDir string, date time.Time) string {
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	return filepath.Join(dataDir, "reviews", y, m, d+".md")
}

// buildMarkdown constructs the full review markdown string from entries and open items.
func buildMarkdown(date time.Time, entries []entry.Entry, todos []scanner.OpenItem, blockers []scanner.OpenItem, cfg config.Config) string {
	var b strings.Builder

	dateStr := date.Format("2006-01-02")
	fmt.Fprintf(&b, "# Daily Review — %s\n", dateStr)

	if len(entries) == 0 && len(todos) == 0 && len(blockers) == 0 {
		b.WriteString("\nNo entries recorded today.\n")
		return b.String()
	}

	// Build a map from entry ID to entry for resolving supersedes references.
	entryByID := make(map[string]entry.Entry, len(entries))
	for _, e := range entries {
		entryByID[e.ID] = e
	}

	// Group today's entries by type.
	grouped := make(map[string][]entry.Entry)
	for _, e := range entries {
		grouped[e.Type] = append(grouped[e.Type], e)
	}

	// Render Achievements section.
	if achs := grouped["achievement"]; len(achs) > 0 {
		b.WriteString("\n## Achievements\n")
		for _, e := range achs {
			project := e.Project
			if project != "" {
				fmt.Fprintf(&b, "- %s (%s)\n", e.Content, project)
			} else {
				fmt.Fprintf(&b, "- %s\n", e.Content)
			}
		}
	}

	// Render Learnings section.
	if lrns := grouped["learning"]; len(lrns) > 0 {
		b.WriteString("\n## Learnings\n")
		for _, e := range lrns {
			fmt.Fprintf(&b, "- %s\n", e.Content)
		}
	}

	// Render Blockers section.
	// For blockers resolved today: check if there's a mutation entry with
	// status=resolved that supersedes a blocker, and show it inline.
	if blks := grouped["blocker"]; len(blks) > 0 {
		b.WriteString("\n## Blockers\n")

		// Build a map of original blocker ID → resolution content for blockers
		// resolved today (via mutation entries in today's log).
		resolutions := buildResolutions(entries)

		for _, e := range blks {
			// Skip mutation entries — they are incorporated inline.
			if e.Supersedes != "" {
				continue
			}
			project := e.Project
			resolution, isResolved := resolutions[e.ID]
			if isResolved {
				if project != "" {
					fmt.Fprintf(&b, "- %s (%s) [resolved: %s]\n", e.Content, project, resolution)
				} else {
					fmt.Fprintf(&b, "- %s [resolved: %s]\n", e.Content, resolution)
				}
			} else {
				if project != "" {
					fmt.Fprintf(&b, "- %s (%s)\n", e.Content, project)
				} else {
					fmt.Fprintf(&b, "- %s\n", e.Content)
				}
			}
		}
	}

	// Render Ideas section.
	if ideas := grouped["idea"]; len(ideas) > 0 {
		b.WriteString("\n## Ideas\n")
		for _, e := range ideas {
			fmt.Fprintf(&b, "- %s\n", e.Content)
		}
	}

	// Render Open Items section — todos then blockers, from scanner.
	if len(todos) > 0 || len(blockers) > 0 {
		b.WriteString("\n## Open Items\n")

		today := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		loc := cfg.Location()
		staleDays := cfg.UI.StaleTriageDays

		// Sort todos: priority order (tomorrow < this-week < someday) then timestamp.
		sortedTodos := make([]scanner.OpenItem, len(todos))
		copy(sortedTodos, todos)
		sort.SliceStable(sortedTodos, func(i, j int) bool {
			pi := priorityOrder[sortedTodos[i].Entry.Priority]
			pj := priorityOrder[sortedTodos[j].Entry.Priority]
			if pi != pj {
				return pi < pj
			}
			return sortedTodos[i].Entry.Timestamp < sortedTodos[j].Entry.Timestamp
		})

		for _, item := range sortedTodos {
			e := item.Entry
			_, overdueDays := scanner.ClassifyItem(item, today, staleDays, loc, cfg.UI.DayStartHour)
			priority := e.Priority
			if priority == "" {
				priority = "someday"
			}
			project := e.Project
			if overdueDays > 0 {
				if project != "" {
					fmt.Fprintf(&b, "- [%s] %s (%s) — %d %s overdue\n",
						priority, e.Content, project, overdueDays, plural("day", overdueDays))
				} else {
					fmt.Fprintf(&b, "- [%s] %s — %d %s overdue\n",
						priority, e.Content, overdueDays, plural("day", overdueDays))
				}
			} else {
				if project != "" {
					fmt.Fprintf(&b, "- [%s] %s (%s)\n", priority, e.Content, project)
				} else {
					fmt.Fprintf(&b, "- [%s] %s\n", priority, e.Content)
				}
			}
		}

		for _, item := range blockers {
			e := item.Entry
			project := e.Project
			if project != "" {
				fmt.Fprintf(&b, "- [blocker] %s (%s)\n", e.Content, project)
			} else {
				fmt.Fprintf(&b, "- [blocker] %s\n", e.Content)
			}
		}
	}

	return b.String()
}

// priorityOrder maps priority string to sort key (lower = higher priority).
var priorityOrder = map[string]int{
	"tomorrow":  0,
	"this-week": 1,
	"someday":   2,
}

// buildResolutions returns a map from original blocker ID to the resolution
// content, built from mutation entries with status="resolved" in today's log.
// It also handles blockers resolved via achievement entries with closes field.
func buildResolutions(entries []entry.Entry) map[string]string {
	resolutions := make(map[string]string)
	for _, e := range entries {
		if e.Type == "blocker" && e.Status == "resolved" && e.Supersedes != "" {
			// This is a resolve-mutation entry. e.Content is the resolution reason.
			resolutions[e.Supersedes] = e.Content
		}
	}
	return resolutions
}

// plural returns word pluralised by appending "s" if count != 1.
func plural(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
