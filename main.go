package main

import (
	"fmt"
	"os"

	"github.com/yourusername/llamasidekick/internal/config"
	"github.com/yourusername/llamasidekick/internal/ui"
)

func main() {
	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Start the UI
	if err := ui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
