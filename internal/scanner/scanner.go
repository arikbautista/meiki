// Package scanner provides the open-item scanner and priority decay logic.
package scanner

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/arikbautista/meiki/internal/dayutil"
	"github.com/arikbautista/meiki/internal/entry"
)

// OpenItem represents an open todo or blocker with its resolved state.
type OpenItem struct {
	Entry       entry.Entry // the original entry (for content, project, tags)
	LatestState entry.Entry // the most recent mutation (for current status)
	AgeDays     int         // days since original capture
}

// entryFilePath returns the path to the JSONL file for the given date
// within an explicit dataDir. Mirrors the logic in entry.EntryFilePath
// but allows callers to specify the root data directory.
func entryFilePath(dataDir string, date time.Time) string {
	y := date.Format("2006")
	m := date.Format("01")
	d := date.Format("2006-01-02")
	return filepath.Join(dataDir, "entries", y, m, d+".jsonl")
}

// readRange reads all entries across the inclusive date range [from, to]
// from an explicit dataDir, skipping missing files silently.
func readRange(dataDir string, from, to time.Time) ([]entry.Entry, error) {
	var result []entry.Entry
	loc := from.Location()
	cur := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc)
	for !cur.After(end) {
		path := entryFilePath(dataDir, cur)
		entries, err := entry.ReadEntriesFromPath(path)
		if err != nil {
			return nil, fmt.Errorf("scanner: read entries for %s: %w", cur.Format("2006-01-02"), err)
		}
		result = append(result, entries...)
		cur = cur.AddDate(0, 0, 1)
	}
	return result, nil
}

// ScanOpenItems scans the JSONL log in dataDir for the most recent scanDays
// days (inclusive of today) and returns open todos and open blockers separately.
//
// Algorithm:
//  1. Collect all entries in the scan window.
//  2. Build originals map (id → entry) for todo/blocker entries.
//  3. Build supersedes map (original_root_id → latest_mutation) by following chains.
//  4. Build closedByAchievement set from achievement entries with closes field.
//  5. An item is open if its terminal-state entry has status "open" and is not
//     in closedByAchievement.
func ScanOpenItems(dataDir string, scanDays int, today time.Time, loc *time.Location, dayStartHour int) (todos []OpenItem, blockers []OpenItem, err error) {
	if scanDays <= 0 {
		scanDays = 30
	}

	from := today.AddDate(0, 0, -(scanDays - 1))
	to := today

	entries, err := readRange(dataDir, from, to)
	if err != nil {
		return nil, nil, err
	}

	// originals: id → entry for all todo/blocker entries that are root entries
	// (i.e. they are not themselves a mutation of something else, or they are the
	// original capture). We track all todo/blocker entries first, then prune
	// entries that are referenced via supersedes as roots.
	originals := make(map[string]entry.Entry)

	// supersedes maps: originalID → latest state entry.
	// "latest" means we process all entries and keep overwriting — since entries
	// are appended in time order, later entries win.
	latestState := make(map[string]entry.Entry)

	// supersedes chain: entryID → the ID it supersedes.
	// Used to walk supersedes chains up to find the root.
	supersedesChain := make(map[string]string)

	// closedByAchievement: set of todo IDs closed by achievement entries.
	closedByAchievement := make(map[string]bool)

	// First pass: collect originals and build the supersedes chain.
	for _, e := range entries {
		switch e.Type {
		case "todo", "blocker":
			if e.Supersedes == "" {
				// This is an original entry (first capture of the item).
				originals[e.ID] = e
			} else {
				// This is a mutation (close/abandon/resolve/reopen mutation entry).
				supersedesChain[e.ID] = e.Supersedes
			}
		case "achievement":
			if e.Closes != "" {
				closedByAchievement[e.Closes] = true
			}
		}
	}

	// Build a helper to find the root original ID by following the supersedes chain.
	// The chain is: mutationC.Supersedes = mutationB.ID, mutationB.Supersedes = originalA.ID
	// findRoot walks from any entry ID to the root original.
	findRoot := func(id string) string {
		visited := make(map[string]bool)
		cur := id
		for {
			if visited[cur] {
				// Cycle guard — should not happen with well-formed data.
				break
			}
			visited[cur] = true
			parent, ok := supersedesChain[cur]
			if !ok {
				// cur has no supersedes pointer; it is a root (or not in chain).
				break
			}
			cur = parent
		}
		return cur
	}

	// Second pass: for each mutation, map it to its root original and track
	// the latest mutation per root.
	for _, e := range entries {
		if (e.Type == "todo" || e.Type == "blocker") && e.Supersedes != "" {
			root := findRoot(e.ID)
			// Only track if we know the root is an original we captured.
			if _, isOriginal := originals[root]; !isOriginal {
				// The root original may be outside the scan window. We still
				// track the mutation so the item is correctly resolved.
				// We cannot produce an OpenItem for it (no original entry),
				// so we skip it.
				continue
			}
			// Keep the latest mutation (entries are in append order, so last wins).
			latestState[root] = e
		}
	}

	// Build the open item lists.
	for id, orig := range originals {
		// Determine the terminal state for this item.
		state, hasMutation := latestState[id]
		if !hasMutation {
			// No mutation found — use the original as its own state.
			state = orig
		}

		// Items with status other than "open" are not open.
		if state.Status != "open" {
			continue
		}

		// Items closed by an achievement are not open.
		if closedByAchievement[id] {
			continue
		}

		// Calculate age from the original timestamp.
		var ageDays int
		ts, parseErr := time.Parse(time.RFC3339, orig.Timestamp)
		if parseErr == nil {
			origDay := dayutil.LogicalDay(ts, loc, dayStartHour)
			ageDays = int(today.Sub(origDay).Hours() / 24)
			if ageDays < 0 {
				ageDays = 0
			}
		}

		item := OpenItem{
			Entry:       orig,
			LatestState: state,
			AgeDays:     ageDays,
		}

		switch orig.Type {
		case "todo":
			todos = append(todos, item)
		case "blocker":
			blockers = append(blockers, item)
		}
	}

	return todos, blockers, nil
}
