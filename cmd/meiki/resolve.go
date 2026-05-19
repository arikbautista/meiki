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

func newResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <id> [how]",
		Short: "Mark an open blocker as resolved",
		Long: `Mark an open blocker as resolved by writing a mutation entry.

The id must reference an existing open blocker. An optional description of how
it was resolved can be provided; if omitted, the content defaults to "resolved".

Example:
  meiki resolve 01ABCDEFGHIJKLMNOPQRSTUVWX
  meiki resolve 01ABCDEFGHIJKLMNOPQRSTUVWX "legal approved the contract"`,
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			how := "resolved"
			if len(args) == 2 && args[1] != "" {
				how = args[1]
			}

			// Ensure the data directory exists.
			if err := config.EnsureDataDir(); err != nil {
				return fmt.Errorf("cannot create data directory: %w", err)
			}

			// Look up the target entry to validate it exists and is an open blocker.
			orig, err := findEntryForResolve(id)
			if err != nil {
				return err
			}

			// Build the mutation entry.
			mut := &entry.Entry{
				ID:         entry.NewID(),
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Type:       "blocker",
				Status:     "resolved",
				Supersedes: id,
				Content:    how,
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

// findEntryForResolve looks up the entry with the given id and validates that
// it is an open blocker. Returns the original entry on success, or a descriptive
// error on failure.
func findEntryForResolve(id string) (entry.Entry, error) {
	dataDir := config.DataDir()

	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		return entry.Entry{}, fmt.Errorf("load config: %w", cfgErr)
	}
	loc := cfg.Location()
	today := dayutil.LogicalDay(time.Now(), loc, cfg.UI.DayStartHour)

	// First check open blockers — the common and fast path.
	_, blockers, err := scanner.ScanOpenItems(dataDir, 90, today, loc, cfg.UI.DayStartHour)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("cannot scan open items: %w", err)
	}
	for _, item := range blockers {
		if item.Entry.ID == id {
			return item.Entry, nil // found and it's open
		}
	}

	// Not in open blockers — search the full history to produce a helpful error.
	all, readErr := entry.ReadEntriesRange(
		today.AddDate(-1, 0, 0),
		today,
	)
	if readErr != nil {
		return entry.Entry{}, fmt.Errorf("cannot read entries: %w", readErr)
	}

	for _, e := range all {
		if e.ID == id {
			if e.Type != "blocker" {
				fmt.Fprintf(os.Stderr, "error: can only resolve blockers\n")
				os.Exit(1)
			}
			// It's a blocker but not open (already resolved, etc.).
			fmt.Fprintf(os.Stderr, "error: blocker is already %s\n", effectiveStatus(e.Status))
			os.Exit(1)
		}
	}

	// Not found at all.
	fmt.Fprintf(os.Stderr, "error: entry not found\n")
	os.Exit(1)

	// Unreachable — os.Exit above always terminates.
	return entry.Entry{}, nil
}
