package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/scanner"
	"github.com/spf13/cobra"
)

// priorityOrder maps priority string to sort key (lower = higher priority).
var priorityOrder = map[string]int{
	"tomorrow":  0,
	"this-week": 1,
	"someday":   2,
}

// openJSONItem is the JSON-serialisable form of an open item.
type openJSONItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Project     string `json:"project,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AgeDays     int    `json:"age_days"`
	OverdueDays int    `json:"overdue_days"`
	Triage      string `json:"triage"`
}

// triageName converts an ItemTriage value to a human-readable string for JSON.
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

// truncateID returns the first 8 characters of id followed by "...".
func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "..."
}

func newOpenCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:          "open",
		Short:        "List open items",
		Long:         "List all open todos and blockers, grouped by type.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			dataDir := config.DataDir()
			todos, blockers, err := scanner.ScanOpenItems(dataDir, cfg.UI.OpenScanDays)
			if err != nil {
				return fmt.Errorf("scan open items: %w", err)
			}

			today := time.Now().UTC()
			today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

			staleDays := cfg.UI.StaleTriageDays

			if jsonOutput {
				return runOpenJSON(cmd, todos, blockers, today, staleDays)
			}
			return runOpenHuman(cmd, todos, blockers, today, staleDays)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON array")
	return cmd
}

// sortTodos sorts todos by priority (tomorrow < this-week < someday) then by
// capture timestamp (earlier first).
func sortTodos(todos []scanner.OpenItem) {
	sort.SliceStable(todos, func(i, j int) bool {
		pi := priorityOrder[todos[i].Entry.Priority]
		pj := priorityOrder[todos[j].Entry.Priority]
		if pi != pj {
			return pi < pj
		}
		// Same priority: earlier capture date first.
		return todos[i].Entry.Timestamp < todos[j].Entry.Timestamp
	})
}

// runOpenHuman prints the human-readable open-item listing.
func runOpenHuman(cmd *cobra.Command, todos, blockers []scanner.OpenItem, today time.Time, staleDays int) error {
	out := cmd.OutOrStdout()

	if len(todos) == 0 && len(blockers) == 0 {
		fmt.Fprintln(out, "No open items.")
		return nil
	}

	sortTodos(todos)

	if len(todos) > 0 {
		fmt.Fprintf(out, "Open Todos (%d):\n", len(todos))
		for _, item := range todos {
			_, overdueDays := scanner.ClassifyItem(item, today, staleDays)
			e := item.Entry
			trunc := truncateID(e.ID)
			project := e.Project
			if project == "" {
				project = "unknown"
			}
			if overdueDays > 0 {
				fmt.Fprintf(out, "  [%s] %-9s %q (%s, %d days overdue)\n",
					trunc, e.Priority, e.Content, project, overdueDays)
			} else {
				fmt.Fprintf(out, "  [%s] %-9s %q (%s)\n",
					trunc, e.Priority, e.Content, project)
			}
		}
	}

	if len(blockers) > 0 {
		fmt.Fprintf(out, "Open Blockers (%d):\n", len(blockers))
		for _, item := range blockers {
			e := item.Entry
			trunc := truncateID(e.ID)
			project := e.Project
			if project == "" {
				project = "unknown"
			}
			fmt.Fprintf(out, "  [%s] %q (%s)\n", trunc, e.Content, project)
		}
	}

	return nil
}

// runOpenJSON prints the JSON-array representation of all open items.
func runOpenJSON(cmd *cobra.Command, todos, blockers []scanner.OpenItem, today time.Time, staleDays int) error {
	out := cmd.OutOrStdout()

	var items []openJSONItem

	for _, item := range todos {
		triage, overdueDays := scanner.ClassifyItem(item, today, staleDays)
		items = append(items, openJSONItem{
			ID:          item.Entry.ID,
			Type:        "todo",
			Content:     item.Entry.Content,
			Project:     item.Entry.Project,
			Priority:    item.Entry.Priority,
			AgeDays:     item.AgeDays,
			OverdueDays: overdueDays,
			Triage:      triageName(triage),
		})
	}

	for _, item := range blockers {
		triage, overdueDays := scanner.ClassifyItem(item, today, staleDays)
		items = append(items, openJSONItem{
			ID:          item.Entry.ID,
			Type:        "blocker",
			Content:     item.Entry.Content,
			Project:     item.Entry.Project,
			AgeDays:     item.AgeDays,
			OverdueDays: overdueDays,
			Triage:      triageName(triage),
		})
	}

	// Emit an empty array (not null) when there are no items.
	if items == nil {
		items = []openJSONItem{}
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}
