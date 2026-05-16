package main

import (
	"fmt"
	"os"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
	"github.com/spf13/cobra"
)

func newAbandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon <id> [reason]",
		Short: "Mark an open todo as abandoned",
		Long: `Mark an open todo as abandoned by writing a mutation entry.

The id must reference an existing open todo. An optional reason can be
provided; if omitted, the reason defaults to "abandoned".

Example:
  meiki abandon 01ABCDEFGHIJKLMNOPQRSTUVWX
  meiki abandon 01ABCDEFGHIJKLMNOPQRSTUVWX "decided not to pursue this"`,
		Args:         cobra.RangeArgs(1, 2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			reason := "abandoned"
			if len(args) == 2 && args[1] != "" {
				reason = args[1]
			}

			// Ensure the data directory exists.
			if err := config.EnsureDataDir(); err != nil {
				return fmt.Errorf("cannot create data directory: %w", err)
			}

			// Look up the target entry to validate it exists and is an open todo.
			orig, err := findEntryForAbandon(id)
			if err != nil {
				return err
			}

			// Build the mutation entry.
			mut := &entry.Entry{
				ID:         entry.NewID(),
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				Type:       "todo",
				Status:     "abandoned",
				Supersedes: id,
				Content:    reason,
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

// findEntryForAbandon looks up the entry with the given id and validates that
// it is an open todo. Returns the original entry on success, or a descriptive
// error on failure.
func findEntryForAbandon(id string) (entry.Entry, error) {
	dataDir := config.DataDir()

	// First check open todos — the common and fast path.
	todos, _, err := scanner.ScanOpenItems(dataDir, 90)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("cannot scan open items: %w", err)
	}
	for _, item := range todos {
		if item.Entry.ID == id {
			return item.Entry, nil // found and it's open
		}
	}

	// Not in open todos — search the full history to produce a helpful error.
	all, readErr := entry.ReadEntriesRange(
		time.Now().UTC().AddDate(-1, 0, 0),
		time.Now().UTC(),
	)
	if readErr != nil {
		return entry.Entry{}, fmt.Errorf("cannot read entries: %w", readErr)
	}

	for _, e := range all {
		if e.ID == id {
			if e.Type != "todo" {
				fmt.Fprintf(os.Stderr, "error: can only abandon todos\n")
				os.Exit(1)
			}
			// It's a todo but not open (abandoned, resolved, etc.).
			fmt.Fprintf(os.Stderr, "error: todo is already %s\n", effectiveStatus(e.Status))
			os.Exit(1)
		}
	}

	// Not found at all.
	fmt.Fprintf(os.Stderr, "error: entry not found\n")
	os.Exit(1)

	// Unreachable — os.Exit above always terminates.
	return entry.Entry{}, nil
}

// effectiveStatus returns a display-friendly status string. Empty strings
// become "unknown" so the error message always reads sensibly.
func effectiveStatus(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
