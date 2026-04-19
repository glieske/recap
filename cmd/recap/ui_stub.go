//go:build !gui

package main

import (
	"fmt"
	"os"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func runUI(cfg *config.Config, store *storage.Store, provider ai.Provider, configPath string, version string) {
	fmt.Fprintln(os.Stderr, "GUI not available — rebuild with: go build -tags gui")
	os.Exit(1)
}
