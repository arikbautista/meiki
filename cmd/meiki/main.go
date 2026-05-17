package main

import (
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

