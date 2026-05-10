// Package cli holds the Cobra command tree for opendelve.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "opendelve",
	Short: "Compliance as code for AI agents",
	Long: `OpenDelve is a GitHub-native compliance bootstrap kit for regulated
AI workflows. It scaffolds a working ISO 9001 quality system in your repo,
generates SOPs, attaches evidence, signs approvals, and emits audit-grade
artifacts. AI agents can read the same machine-readable policies and use
them to verify your software stays compliant.

Repository is the source of truth. Local-first. No telemetry. MIT licensed.`,
}

// Execute runs the root command. version/commit/date are injected at build time.
func Execute(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
