package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/dayutil"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/spf13/cobra"
)

func newRecentCmd() *cobra.Command {
	var days int
	var typeFilter string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:          "recent",
		Short:        "Show recent entries",
		Long:         "Show entries from the last N days, grouped by date and type.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate --type flag if provided.
			if typeFilter != "" && !entry.ValidTypes[typeFilter] {
				types := make([]string, 0, len(entry.ValidTypes))
				for t := range entry.ValidTypes {
					types = append(types, t)
				}
				sort.Strings(types)
				return fmt.Errorf("invalid type %q: must be one of %s", typeFilter, strings.Join(types, ", "))
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			loc := cfg.Location()
			today := dayutil.LogicalDay(time.Now(), loc, cfg.UI.DayStartHour)
			from := today.AddDate(0, 0, -(days - 1))

			entries, err := entry.ReadEntriesRange(from, today)
			if err != nil {
				return fmt.Errorf("read entries: %w", err)
			}

			// Filter by type if specified.
			if typeFilter != "" {
				filtered := entries[:0]
				for _, e := range entries {
					if e.Type == typeFilter {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}

			if jsonOutput {
				return runRecentJSON(cmd, entries)
			}
			return runRecentHuman(cmd, entries, days, loc, cfg.UI.DayStartHour)
		},
	}

	cmd.Flags().IntVar(&days, "days", 7, "Number of days to look back")
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter to a single entry type")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON array")
	return cmd
}

// runRecentHuman prints entries grouped by date (most recent first) then by
// type within each date in human-readable form.
func runRecentHuman(cmd *cobra.Command, entries []entry.Entry, days int, loc *time.Location, dayStartHour int) error {
	out := cmd.OutOrStdout()

	if len(entries) == 0 {
		fmt.Fprintf(out, "No entries in the last %d days.\n", days)
		return nil
	}

	// Group entries by date string (YYYY-MM-DD).
	byDate := make(map[string][]entry.Entry)
	for _, e := range entries {
		date := dateKey(e.Timestamp, loc, dayStartHour)
		byDate[date] = append(byDate[date], e)
	}

	// Collect and sort date keys, most recent first.
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, date := range dates {
		fmt.Fprintf(out, "%s:\n", date)

		dateEntries := byDate[date]

		// Group by type.
		grouped := make(map[string][]entry.Entry)
		for _, e := range dateEntries {
			grouped[e.Type] = append(grouped[e.Type], e)
		}

		for _, typeName := range typeGroupOrder {
			group := grouped[typeName]
			if len(group) == 0 {
				continue
			}

			label := typeGroupLabel[typeName]
			fmt.Fprintf(out, "  %s (%d):\n", label, len(group))

			for _, e := range group {
				trunc := truncateID(e.ID)
				project := e.Project
				if project == "" {
					project = "unknown"
				}

				if e.Supersedes != "" {
					mutLabel := mutationLabel(e.Status)
					fmt.Fprintf(out, "    [%s] [%s] %q (%s)\n", trunc, mutLabel, e.Content, project)
				} else if typeName == "todo" && e.Priority != "" {
					fmt.Fprintf(out, "    [%s] %s %q (%s)\n", trunc, e.Priority, e.Content, project)
				} else {
					fmt.Fprintf(out, "    [%s] %q (%s)\n", trunc, e.Content, project)
				}
			}
		}
	}

	return nil
}

// runRecentJSON prints entries as a JSON array sorted by timestamp descending.
func runRecentJSON(cmd *cobra.Command, entries []entry.Entry) error {
	out := cmd.OutOrStdout()

	// Sort by timestamp descending.
	sorted := make([]entry.Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Timestamp > sorted[j].Timestamp
	})

	// Emit an empty array (not null) when there are no entries.
	if sorted == nil {
		sorted = []entry.Entry{}
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(sorted)
}

// dateKey extracts the logical YYYY-MM-DD date string from an RFC3339 timestamp.
// If parsing fails, it returns the first 10 characters as a best-effort fallback.
func dateKey(ts string, loc *time.Location, dayStartHour int) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		if len(ts) >= 10 {
			return ts[:10]
		}
		return ts
	}
	return dayutil.LogicalDayStr(t, loc, dayStartHour)
}
