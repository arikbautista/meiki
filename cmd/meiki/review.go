package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/arikbautista/meiki/internal/review"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	var silent bool

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Generate today's daily review",
		Long: `Generate today's daily review as a markdown file.

The review is written to <data_dir>/reviews/YYYY/MM/YYYY-MM-DD.md and
includes achievements, learnings, blockers, ideas, and open items.

Running review multiple times on the same day overwrites the file with
current data (idempotent).`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			dataDir := config.DataDir()
			now := time.Now().UTC()

			// Generate the review markdown.
			md, err := review.GenerateReview(dataDir, now, cfg)
			if err != nil {
				return fmt.Errorf("generate review: %w", err)
			}

			// Determine the output path and create intermediate directories.
			reviewPath := review.ReviewFilePath(dataDir, now)
			if err := os.MkdirAll(filepath.Dir(reviewPath), 0o755); err != nil {
				return fmt.Errorf("create review directory: %w", err)
			}

			// Write the review file (overwrite if exists — idempotent).
			if err := os.WriteFile(reviewPath, []byte(md), 0o644); err != nil {
				return fmt.Errorf("write review file: %w", err)
			}

			// Update last_review_ts in state.json.
			if err := config.UpdateReviewTS(now); err != nil {
				return fmt.Errorf("update review timestamp: %w", err)
			}

			// Print path and content to stdout unless --silent.
			if !silent {
				fmt.Fprintln(cmd.OutOrStdout(), reviewPath)
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprint(cmd.OutOrStdout(), md)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&silent, "silent", false, "Suppress stdout output (file is still written)")
	return cmd
}
