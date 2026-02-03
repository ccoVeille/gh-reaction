// Package main implements a CLI tool to analyze GitHub reactions on your posts (issues, PRs, comments).
package main

import (
	"context"
	_ "embed"
	"os"
	"os/signal"

	"github.com/ccoVeille/gh-reaction/internal/cmd"
)

func run() error {
	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	rootCmd := cmd.NewRootCmd()

	return rootCmd.ExecuteContext(ctx)
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
