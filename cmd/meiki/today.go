package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/dayutil"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/spf13/cobra"
)

// typeGroupOrder defines the display order for entry types in today's output.
var typeGroupOrder = []string{"achievement", "learning", "blocker", "todo", "idea"}

// typeGroupLabel maps singular entry type to a plural display label.
var typeGroupLabel = map[string]string{
	"achievement": "Achievements",
	"learning":    "Learnings",
	"blocker":     "Blockers",
	"todo":        "Todos",
	"idea":        "Ideas",
}

func newTodayCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:          "today",
		Short:        "Show today's entries",
		Long:         "Show all entries captured today, grouped by type.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			loc := cfg.Location()
			today := dayutil.LogicalDay(time.Now(), loc, cfg.UI.DayStartHour)
			entries, err := entry.ReadEntries(today)
			if err != nil {
				return fmt.Errorf("read today's entries: %w", err)
			}

			if jsonOutput {
				return runTodayJSON(cmd, entries)
			}
			return runTodayHuman(cmd, entries)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON array")
	return cmd
}

// runTodayHuman prints today's entries grouped by type in human-readable form.
func runTodayHuman(cmd *cobra.Command, entries []entry.Entry) error {
	out := cmd.OutOrStdout()

	if len(entries) == 0 {
		fmt.Fprintln(out, "Nothing logged today.")
		return nil
	}

	// Group entries by type.
	grouped := make(map[string][]entry.Entry)
	for _, e := range entries {
		grouped[e.Type] = append(grouped[e.Type], e)
	}

	for _, typeName := range typeGroupOrder {
		group := grouped[typeName]
		if len(group) == 0 {
			continue
		}

		label := typeGroupLabel[typeName]
		fmt.Fprintf(out, "%s (%d):\n", label, len(group))

		for _, e := range group {
			trunc := truncateID(e.ID)
			project := e.Project
			if project == "" {
				project = "unknown"
			}

			if e.Supersedes != "" {
				// Mutation entry: show the status label and content as reason.
				mutLabel := mutationLabel(e.Status)
				fmt.Fprintf(out, "  [%s] [%s] %q (%s)\n", trunc, mutLabel, e.Content, project)
			} else if typeName == "todo" && e.Priority != "" {
				// Todo with priority: show the priority.
				fmt.Fprintf(out, "  [%s] %s %q (%s)\n", trunc, e.Priority, e.Content, project)
			} else {
				fmt.Fprintf(out, "  [%s] %q (%s)\n", trunc, e.Content, project)
			}
		}
	}

	return nil
}

// mutationLabel returns a human-friendly label for a mutation entry based on status.
func mutationLabel(status string) string {
	switch strings.ToLower(status) {
	case "abandoned":
		return "abandoned"
	case "resolved":
		return "resolved"
	case "open":
		return "reopened"
	default:
		if status != "" {
			return status
		}
		return "mutated"
	}
}

// runTodayJSON prints today's entries as a raw JSON array.
func runTodayJSON(cmd *cobra.Command, entries []entry.Entry) error {
	out := cmd.OutOrStdout()

	// Emit an empty array (not null) when there are no entries.
	if entries == nil {
		entries = []entry.Entry{}
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}
