// Package main is the OpenDelve CLI entry point.
package main

import (
	"github.com/lloydarmbrust/opendelve/internal/cli"
)

// Set by GoReleaser at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Execute(version, commit, date)
}
