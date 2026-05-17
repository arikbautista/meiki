package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/arikbautista/meiki/internal/config"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify meiki installation health",
		Long:  `Run diagnostic checks to verify that meiki is correctly installed and configured.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			issues := 0

			dataDir := config.DataDir()
			configDir := config.ConfigDir()

			// Check 1: Data directory exists and is writable
			if info, err := os.Stat(dataDir); err != nil || !info.IsDir() {
				fmt.Fprintf(w, "✗ Data directory: %s — missing or not a directory\n", dataDir)
				issues++
			} else {
				// Check writability by attempting to create a temp file
				tmp, err := os.CreateTemp(dataDir, ".doctor-check-*")
				if err != nil {
					fmt.Fprintf(w, "✗ Data directory: %s — not writable\n", dataDir)
					issues++
				} else {
					tmp.Close()
					os.Remove(tmp.Name())
					fmt.Fprintf(w, "✓ Data directory: %s\n", dataDir)
				}
			}

			// Check 2: entries/ subdirectory exists
			entriesDir := filepath.Join(dataDir, "entries")
			if info, err := os.Stat(entriesDir); err != nil || !info.IsDir() {
				fmt.Fprintf(w, "✗ Entries directory: missing — run 'meiki setup' to create\n")
				issues++
			} else {
				fmt.Fprintf(w, "✓ Entries directory: exists\n")
			}

			// Check 3: reviews/ subdirectory exists
			reviewsDir := filepath.Join(dataDir, "reviews")
			if info, err := os.Stat(reviewsDir); err != nil || !info.IsDir() {
				fmt.Fprintf(w, "✗ Reviews directory: missing — run 'meiki setup' to create\n")
				issues++
			} else {
				fmt.Fprintf(w, "✓ Reviews directory: exists\n")
			}

			// Check 4: state.json is valid (if it exists)
			stateFile := filepath.Join(dataDir, "state.json")
			if _, err := os.Stat(stateFile); os.IsNotExist(err) {
				fmt.Fprintf(w, "✓ State file: not present (OK)\n")
			} else if err != nil {
				fmt.Fprintf(w, "✗ State file: cannot access — %v\n", err)
				issues++
			} else {
				data, err := os.ReadFile(stateFile)
				if err != nil {
					fmt.Fprintf(w, "✗ State file: cannot read — %v\n", err)
					issues++
				} else {
					var js json.RawMessage
					if err := json.Unmarshal(data, &js); err != nil {
						fmt.Fprintf(w, "✗ State file: malformed JSON — %v\n", err)
						issues++
					} else {
						fmt.Fprintf(w, "✓ State file: valid\n")
					}
				}
			}

			// Check 5: Config file parses (if it exists)
			configFile := filepath.Join(configDir, "config.toml")
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Fprintf(w, "✓ Config file: not present (using defaults)\n")
			} else {
				if _, err := config.LoadConfig(); err != nil {
					fmt.Fprintf(w, "✗ Config file: parse error — %v\n", err)
					issues++
				} else {
					fmt.Fprintf(w, "✓ Config file: valid\n")
				}
			}

			// Check 6: meiki binary on PATH
			if path, err := exec.LookPath("meiki"); err != nil {
				fmt.Fprintf(w, "✗ Binary on PATH: not found\n")
				issues++
			} else {
				fmt.Fprintf(w, "✓ Binary on PATH: %s\n", path)
			}

			// Summary
			fmt.Fprintln(w)
			if issues == 0 {
				fmt.Fprintln(w, "All checks passed.")
				return nil
			}

			noun := "issue"
			if issues > 1 {
				noun = "issues"
			}
			msg := fmt.Sprintf("%d %s found. Run 'meiki setup' to fix.", issues, noun)
			fmt.Fprintln(w, msg)
			return fmt.Errorf("%s", msg)
		},
	}
	return cmd
}
