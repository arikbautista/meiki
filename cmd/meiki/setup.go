package main

import (
	"fmt"
	"os"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initialize meiki and print integration snippets",
		Long: `Initialize meiki data and config directories, then print integration
snippets for AI tool configuration.

This command is idempotent and safe to re-run at any time.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Create data directories.
			if err := config.EnsureDataDir(); err != nil {
				return fmt.Errorf("create data directories: %w", err)
			}

			// 2. Create config directory.
			if err := os.MkdirAll(config.ConfigDir(), 0o755); err != nil {
				return fmt.Errorf("create config directory: %w", err)
			}

			out := cmd.OutOrStdout()

			// 3. Success message.
			fmt.Fprintln(out, "meiki initialized successfully.")
			fmt.Fprintln(out)

			// 4. Print MEIKI.md content with instructions.
			fmt.Fprintln(out, "--- MEIKI.md (paste into ~/.claude/CLAUDE.md) ---")
			fmt.Fprintln(out)
			fmt.Fprint(out, meikiMD)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "--- end MEIKI.md ---")
			fmt.Fprintln(out)

			// 5. Print Stop hook snippet.
			fmt.Fprintln(out, "--- Stop hook (optional, paste into ~/.claude/settings.json) ---")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "This hook auto-runs `meiki review` when a session ends.")
			fmt.Fprintln(out, "Optional if MEIKI.md is already in your CLAUDE.md — the AI will")
			fmt.Fprintln(out, "run `meiki review` on session end via the lifecycle triggers.")
			fmt.Fprintln(out)
			fmt.Fprintln(out, stopHookSnippet)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "--- end Stop hook ---")
			fmt.Fprintln(out)

			// 6. Reminder.
			fmt.Fprintln(out, "Run `meiki doctor` to verify your installation.")

			return nil
		},
	}
	return cmd
}
