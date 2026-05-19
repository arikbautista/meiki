package main

import (
	"fmt"
	"os"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/dayutil"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
	"github.com/spf13/cobra"
)

func newReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <id>",
		Short: "Reopen an abandoned, resolved, or closed item",
		Long: `Reopen an abandoned, resolved, or closed todo or blocker by writing a mutation
entry that restores open status.

The id must reference an existing non-open todo or blocker. The id may point to
the original entry or to any mutation in its chain; meiki will locate the latest
state-bearing mutation and link from there.

Example:
  meiki reopen 01ABCDEFGHIJKLMNOPQRSTUVWX`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// Ensure the data directory exists.
			if err := config.EnsureDataDir(); err != nil {
				return fmt.Errorf("cannot create data directory: %w", err)
			}

			// Look up the target entry, validate it is a closed todo or blocker,
			// and resolve the latest mutation in the supersedes chain.
			orig, latestID, err := findEntryForReopen(id)
			if err != nil {
				return err
			}

			// Build the mutation entry. supersedes points at the latest mutation
			// in the chain (which may be id itself if there are no further mutations).
			mut := &entry.Entry{
				ID:         entry.NewID(),
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Type:       orig.Type,
				Status:     "open",
				Supersedes: latestID,
				Content:    "reopened",
				Source:     "cli",
				Project:    orig.Project,
			}

			mutID, err := entry.AppendEntry(mut)
			if err != nil {
				return fmt.Errorf("write mutation entry: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), mutID)
			return nil
		},
	}
}

// findEntryForReopen locates the entry for the given id and validates it is a
// closed (non-open) todo or blocker. Returns the original entry (for type and
// project), the ID of the latest state-bearing mutation in the supersedes chain
// (the new reopen entry's supersedes field should point here), and any error.
func findEntryForReopen(id string) (orig entry.Entry, latestID string, err error) {
	dataDir := config.DataDir()

	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		return entry.Entry{}, "", fmt.Errorf("load config: %w", cfgErr)
	}
	loc := cfg.Location()
	today := dayutil.LogicalDay(time.Now(), loc, cfg.UI.DayStartHour)

	// Check whether the item is currently open — reopen is only valid on closed items.
	todos, blockers, scanErr := scanner.ScanOpenItems(dataDir, 90, today, loc, cfg.UI.DayStartHour)
	if scanErr != nil {
		return entry.Entry{}, "", fmt.Errorf("cannot scan open items: %w", scanErr)
	}
	for _, item := range todos {
		if item.Entry.ID == id || item.LatestState.ID == id {
			fmt.Fprintf(os.Stderr, "error: already open\n")
			os.Exit(1)
		}
	}
	for _, item := range blockers {
		if item.Entry.ID == id || item.LatestState.ID == id {
			fmt.Fprintf(os.Stderr, "error: already open\n")
			os.Exit(1)
		}
	}

	// Read all entries to locate the target and build the supersedes chain.
	all, readErr := entry.ReadEntriesRange(
		today.AddDate(-1, 0, 0),
		today,
	)
	if readErr != nil {
		return entry.Entry{}, "", fmt.Errorf("cannot read entries: %w", readErr)
	}

	// Find the entry with the given ID.
	found := false
	var target entry.Entry
	for _, e := range all {
		if e.ID == id {
			target = e
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "error: entry not found\n")
		os.Exit(1)
	}

	// Validate type: only todo and blocker support status mutations.
	if target.Type != "todo" && target.Type != "blocker" {
		fmt.Fprintf(os.Stderr, "error: cannot reopen %s\n", target.Type)
		os.Exit(1)
	}

	// Resolve the root original entry by following the supersedes chain backward.
	// Build a map of id → entry for fast lookup.
	byID := make(map[string]entry.Entry, len(all))
	for _, e := range all {
		byID[e.ID] = e
	}

	// Walk the supersedes chain from target to the root original.
	rootID := target.ID
	visited := make(map[string]bool)
	cur := target
	for cur.Supersedes != "" && !visited[cur.ID] {
		visited[cur.ID] = true
		parent, ok := byID[cur.Supersedes]
		if !ok {
			// Parent may be outside the scan window; stop here.
			break
		}
		cur = parent
		rootID = cur.ID
	}
	rootEntry := byID[rootID]

	// Find the latest mutation in the chain by scanning forward. The latest
	// mutation is the entry with a supersedes pointer into this chain that
	// appears latest in the log (entries are appended in time order).
	// We identify all IDs belonging to this chain.
	chainIDs := make(map[string]bool)
	chainIDs[rootID] = true
	// Include all entries that supersede something in the chain.
	changed := true
	for changed {
		changed = false
		for _, e := range all {
			if chainIDs[e.ID] {
				continue
			}
			if chainIDs[e.Supersedes] {
				chainIDs[e.ID] = true
				changed = true
			}
		}
	}

	// The latest chain member is the one that appears last in the log and
	// is not superseded by any other chain member.
	supersededByChain := make(map[string]bool)
	for _, e := range all {
		if chainIDs[e.ID] && e.Supersedes != "" && chainIDs[e.Supersedes] {
			supersededByChain[e.Supersedes] = true
		}
	}

	// The terminal entry is the chain member not superseded by another chain member.
	terminalID := rootID
	for _, e := range all {
		if chainIDs[e.ID] && !supersededByChain[e.ID] {
			terminalID = e.ID
		}
	}

	return rootEntry, terminalID, nil
}
