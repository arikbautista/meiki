package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/dayutil"
	"github.com/arikbautista/meiki/internal/entry"
	"github.com/arikbautista/meiki/internal/scanner"
	"github.com/spf13/cobra"
)

func newCaptureCmd() *cobra.Command {
	var (
		project     string
		tags        string
		priority    string
		due         string
		closes      string
		externalRef string
	)

	cmd := &cobra.Command{
		Use:   "capture <type> <content>",
		Short: "Capture a work entry",
		Long: `Capture a work entry of the given type.

Types: achievement, learning, blocker, todo, idea

Examples:
  meiki capture todo "finish the report"
  meiki capture achievement "shipped the feature" --closes <todo-id>
  meiki capture blocker "waiting on legal approval" --project myapp
  meiki capture todo "review PR" --priority tomorrow --tags review,code`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			entryType := args[0]
			content := args[1]

			// Validate entry type early for a clear error message.
			if !entry.ValidTypes[entryType] {
				return fmt.Errorf("invalid type %q: must be one of achievement, learning, blocker, todo, idea", entryType)
			}

			// Type-specific flag validation.
			if priority != "" && entryType != "todo" {
				return fmt.Errorf("--priority is only allowed on todo entries, got type %q", entryType)
			}
			if due != "" && entryType != "todo" {
				return fmt.Errorf("--due is only allowed on todo entries, got type %q", entryType)
			}
			if closes != "" && entryType != "achievement" {
				return fmt.Errorf("--closes is only allowed on achievement entries, got type %q", entryType)
			}

			// Validate due date format.
			if due != "" {
				if _, err := time.Parse("2006-01-02", due); err != nil {
					return fmt.Errorf("--due must be in YYYY-MM-DD format, got %q", due)
				}
			}

			// Auto-detect project from cwd basename if not provided.
			if project == "" {
				if cwd, err := os.Getwd(); err == nil {
					project = filepath.Base(cwd)
				}
			}

			// Ensure the data directory exists.
			if err := config.EnsureDataDir(); err != nil {
				return fmt.Errorf("cannot create data directory: %w", err)
			}

			// Validate --closes references an existing open todo.
			if closes != "" {
				if err := validateCloses(closes); err != nil {
					return fmt.Errorf("--closes: %w", err)
				}
			}

			// Build the entry.
			e, err := entry.NewEntry(entryType, content)
			if err != nil {
				return err
			}

			e.Source = "cli"
			e.Project = project

			if tags != "" {
				e.Tags = parseTags(tags)
			}

			if externalRef != "" {
				e.ExternalRef = externalRef
			}

			// Apply type-specific defaults and fields.
			switch entryType {
			case "todo":
				if priority == "" {
					priority = "this-week"
				}
				e.Priority = priority
				if due != "" {
					e.Due = due
				}
				e.Status = "open"
			case "blocker":
				e.Status = "open"
			case "achievement":
				if closes != "" {
					e.Closes = closes
				}
			}

			// Re-validate the fully populated entry.
			if err := entry.Validate(e); err != nil {
				return err
			}

			id, err := entry.AppendEntry(e)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), id)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project/repo context (defaults to cwd basename)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags (e.g. review,code)")
	cmd.Flags().StringVar(&priority, "priority", "", "Priority level: tomorrow|this-week|someday (todo only)")
	cmd.Flags().StringVar(&due, "due", "", "Due date in YYYY-MM-DD format (todo only)")
	cmd.Flags().StringVar(&closes, "closes", "", "ID of the todo this achievement completes (achievement only)")
	cmd.Flags().StringVar(&externalRef, "external-ref", "", "External reference (e.g. jira:ENG-1234)")

	return cmd
}

// parseTags splits a comma-separated tag string into a trimmed slice,
// dropping any empty elements.
func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// validateCloses checks that the given id refers to an existing open todo
// within the configured scan window. It uses ScanOpenItems which reads from
// the real data directory.
func validateCloses(id string) error {
	dataDir := config.DataDir()

	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		return fmt.Errorf("load config: %w", cfgErr)
	}
	loc := cfg.Location()
	today := dayutil.LogicalDay(time.Now(), loc, cfg.UI.DayStartHour)

	// Use a generous scan window (90 days) so we find older todos.
	todos, _, err := scanner.ScanOpenItems(dataDir, 90, today, loc, cfg.UI.DayStartHour)
	if err != nil {
		return fmt.Errorf("cannot scan open items: %w", err)
	}

	for _, item := range todos {
		if item.Entry.ID == id {
			return nil // found and it's open
		}
	}

	// Not found in open todos — check if it exists at all via a broader read.
	// If it doesn't exist in open items it is either non-existent or not an open todo.
	all, readErr := entry.ReadEntriesRange(
		today.AddDate(-1, 0, 0),
		today,
	)
	if readErr != nil {
		return fmt.Errorf("cannot read entries: %w", readErr)
	}

	for _, e := range all {
		if e.ID == id {
			if e.Type != "todo" {
				return fmt.Errorf("entry %q exists but is type %q, not todo", id, e.Type)
			}
			return fmt.Errorf("entry %q exists but is not an open todo (status: %q)", id, e.Status)
		}
	}

	return fmt.Errorf("entry %q does not exist", id)
}
