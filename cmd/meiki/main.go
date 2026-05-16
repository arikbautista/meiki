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

func newBriefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brief",
		Short: "Show the morning briefing",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Generate a daily review",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "List open items",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newTodayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "Show today's entries",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newRecentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recent",
		Short: "Show recent entries",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newAbandonCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "abandon",
		Short: "Mark an open item as abandoned",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve",
		Short: "Mark an open item as resolved",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
}

func newReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen",
		Short: "Reopen an abandoned or resolved item",
		Run:   func(cmd *cobra.Command, args []string) { notImplemented(cmd) },
	}
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
