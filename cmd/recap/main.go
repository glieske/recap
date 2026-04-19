package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
	"github.com/glieske/recap/internal/tui"
)

var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

func main() {
	configLong := flag.String("config", "", "override config file path")
	configShort := flag.String("c", "", "override config file path")
	versionLong := flag.Bool("version", false, "print version and exit")
	versionShort := flag.Bool("v", false, "print version and exit")
	autoNew := false
	runGUI := false
	if len(os.Args) > 1 && os.Args[1] == "new" {
		autoNew = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	if len(os.Args) > 1 && os.Args[1] == "ui" {
		runGUI = true
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

	if runGUI {
		runUI(cfg, store, provider, configPath, version)
		return
	}

	app := tui.NewAppModel(cfg, store, provider, configPath, autoNew, version)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
