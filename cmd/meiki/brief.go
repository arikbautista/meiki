package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/arikbautista/meiki/internal/brief"
	"github.com/arikbautista/meiki/internal/config"
	"github.com/spf13/cobra"
)

func newBriefCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "brief",
		Short: "Show the morning briefing",
		Long: `Show the morning briefing with open todos, blockers, and items needing triage.

The briefing is debounced: if it has already been produced today with no new
entries since, it outputs nothing and exits 0.

On a fresh install with no history, a welcome message is printed instead.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			dataDir := config.DataDir()

			state, err := config.LoadState()
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}

			b, err := brief.GenerateBriefing(dataDir, cfg, state)
			if err != nil {
				return fmt.Errorf("generate briefing: %w", err)
			}

			// Debounced: output nothing, exit 0.
			if b.Debounced {
				return nil
			}

			// Welcome: print welcome message, exit 0.
			if b.Welcome {
				fmt.Fprintln(cmd.OutOrStdout(), brief.FormatHuman(b))
				return nil
			}

			// Normal briefing output.
			if jsonOutput {
				type jsonBriefing struct {
					ReviewSummary string           `json:"review_summary"`
					OpenTodos     []brief.BriefItem `json:"open_todos"`
					OpenBlockers  []brief.BriefItem `json:"open_blockers"`
					NeedsTriage   []brief.BriefItem `json:"needs_triage"`
				}
				out := jsonBriefing{
					ReviewSummary: b.ReviewSummary,
					OpenTodos:     b.OpenTodos,
					OpenBlockers:  b.OpenBlockers,
					NeedsTriage:   b.NeedsTriage,
				}
				// Ensure slices are non-nil in JSON output.
				if out.OpenTodos == nil {
					out.OpenTodos = []brief.BriefItem{}
				}
				if out.OpenBlockers == nil {
					out.OpenBlockers = []brief.BriefItem{}
				}
				if out.NeedsTriage == nil {
					out.NeedsTriage = []brief.BriefItem{}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(out); err != nil {
					return fmt.Errorf("encode json: %w", err)
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), brief.FormatHuman(b))
			}

			// Update last_brief_ts after producing output.
			if err := config.UpdateBriefTS(time.Now().UTC()); err != nil {
				return fmt.Errorf("update brief timestamp: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output briefing as JSON")
	return cmd
}
