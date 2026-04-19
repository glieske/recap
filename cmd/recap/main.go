package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
	"github.com/glieske/recap/internal/tui"
)

var version = "dev"

func main() {
	configLong := flag.String("config", "", "override config file path")
	configShort := flag.String("c", "", "override config file path")
	versionLong := flag.Bool("version", false, "print version and exit")
	versionShort := flag.Bool("v", false, "print version and exit")
	autoNew := false
	if len(os.Args) > 1 && os.Args[1] == "new" {
		autoNew = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	flag.Parse()

	if *versionLong || *versionShort {
		fmt.Printf("recap version %s\n", version)
		return
	}

	configPath := *configLong
	if configPath == "" {
		configPath = *configShort
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	store := storage.NewStore(cfg.NotesDir)

	provider, err := ai.NewProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: AI provider unavailable: %v\n", err)
		provider = nil
	}

	app := tui.NewAppModel(cfg, store, provider, configPath, autoNew)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
