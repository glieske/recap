//go:build gui

package main

import (
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/gui"
	"github.com/glieske/recap/internal/storage"
)

func runUI(cfg *config.Config, store *storage.Store, provider ai.Provider, configPath string, version string) {
	gui.Run(cfg, store, provider, configPath, version)
}
