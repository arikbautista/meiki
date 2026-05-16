package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "meiki",
		Short:        "Work memory for AI CLI sessions",
		Long:         "meiki captures work memory during AI CLI sessions, produces daily reviews, and delivers next-morning briefings.",
		SilenceUsage: true,
		Version:      version,
	}

	root.AddCommand(
		newCaptureCmd(),
		newBriefCmd(),
		newReviewCmd(),
		newOpenCmd(),
		newTodayCmd(),
		newRecentCmd(),
		newAbandonCmd(),
		newResolveCmd(),
		newReopenCmd(),
		newSetupCmd(),
		newDoctorCmd(),
	)

	return root
}

func notImplemented(cmd *cobra.Command) {
	fmt.Fprintf(os.Stderr, "meiki %s: not implemented\n", cmd.Name())
	os.Exit(1)
}

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Initialize meiki configuration",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose configuration and data directory issues",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}
