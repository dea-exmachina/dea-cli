package main

import (
	"github.com/dea-exmachina/dea-cli/internal/commands"
)

// Injected at build time by GoReleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	commands.Execute(version, commit, date)
}
