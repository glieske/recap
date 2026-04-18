package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPathContainsRecap(t *testing.T) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	if !strings.Contains(configPath, "recap") {
		t.Fatalf("DefaultConfigPath() = %q, want path containing %q", configPath, "recap")
	}

	notesSegment := string(filepath.Separator) + "notes" + string(filepath.Separator)
	if strings.Contains(configPath, notesSegment) {
		t.Fatalf("DefaultConfigPath() = %q, should not contain legacy segment %q", configPath, notesSegment)
	}
}

func TestDefaultConfigReturnsRecapNotesDir(t *testing.T) {
	cfg, err := defaultConfig()
	if err != nil {
		t.Fatalf("defaultConfig() error = %v", err)
	}

	if filepath.Base(cfg.NotesDir) != "recap" {
		t.Fatalf("defaultConfig().NotesDir = %q, want path ending with %q", cfg.NotesDir, "recap")
	}
}

func TestApplyDefaultsUsesRecapFallback(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg := &Config{NotesDir: ""}
	if err := applyDefaultsAndNormalize(cfg); err != nil {
		t.Fatalf("applyDefaultsAndNormalize() error = %v", err)
	}

	if filepath.Base(cfg.NotesDir) != "recap" {
		t.Fatalf("applyDefaultsAndNormalize().NotesDir = %q, want path ending with %q", cfg.NotesDir, "recap")
	}
}

func TestMigrateOldConfig_NoOldFile(t *testing.T) {
	xdgConfigHome := t.TempDir()

	message, err := migrateOldConfigWithDir(xdgConfigHome)
	if err != nil {
		t.Fatalf("MigrateOldConfig() error = %v", err)
	}

	if message != "" {
		t.Fatalf("MigrateOldConfig() message = %q, want empty message when old config does not exist", message)
	}

	newConfigPath := filepath.Join(xdgConfigHome, "recap", "config.yaml")
	if _, err := os.Stat(newConfigPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no new config file at %q, stat error = %v", newConfigPath, err)
	}
}

func TestMigrateOldConfig_OldExistsNewDoesNot(t *testing.T) {
	xdgConfigHome := t.TempDir()

	oldConfigPath := filepath.Join(xdgConfigHome, "notes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(oldConfigPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(old config dir) error = %v", err)
	}

	oldConfigData := []byte("notes_dir: ~/recap\nai_provider: github_models\n")
	if err := os.WriteFile(oldConfigPath, oldConfigData, 0o600); err != nil {
		t.Fatalf("WriteFile(old config) error = %v", err)
	}

	message, err := migrateOldConfigWithDir(xdgConfigHome)
	if err != nil {
		t.Fatalf("MigrateOldConfig() error = %v", err)
	}

	if message == "" {
		t.Fatalf("MigrateOldConfig() message = empty, want migration notification")
	}

	newConfigPath := filepath.Join(xdgConfigHome, "recap", "config.yaml")
	newConfigData, err := os.ReadFile(newConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(new config) error = %v", err)
	}

	if string(newConfigData) != string(oldConfigData) {
		t.Fatalf("migrated config content mismatch: got %q, want %q", string(newConfigData), string(oldConfigData))
	}
}

func TestMigrateOldConfig_NewAlreadyExists(t *testing.T) {
	xdgConfigHome := t.TempDir()

	oldConfigPath := filepath.Join(xdgConfigHome, "notes", "config.yaml")
	newConfigPath := filepath.Join(xdgConfigHome, "recap", "config.yaml")

	if err := os.MkdirAll(filepath.Dir(oldConfigPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(old config dir) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(newConfigPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(new config dir) error = %v", err)
	}

	oldConfigData := []byte("notes_dir: ~/legacy\n")
	existingNewData := []byte("notes_dir: ~/recap\n")

	if err := os.WriteFile(oldConfigPath, oldConfigData, 0o600); err != nil {
		t.Fatalf("WriteFile(old config) error = %v", err)
	}
	if err := os.WriteFile(newConfigPath, existingNewData, 0o600); err != nil {
		t.Fatalf("WriteFile(new config) error = %v", err)
	}

	message, err := migrateOldConfigWithDir(xdgConfigHome)
	if err != nil {
		t.Fatalf("MigrateOldConfig() error = %v", err)
	}

	if message != "" {
		t.Fatalf("MigrateOldConfig() message = %q, want empty when new config already exists", message)
	}

	actualNewData, err := os.ReadFile(newConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(new config) error = %v", err)
	}

	if string(actualNewData) != string(existingNewData) {
		t.Fatalf("new config should remain unchanged: got %q, want %q", string(actualNewData), string(existingNewData))
	}
}

func TestMigrateOldConfig_PublicWrapperMigratesUsingUserConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error = %v", err)
	}

	oldConfigPath := filepath.Join(userConfigDir, "notes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(oldConfigPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(old config dir) error = %v", err)
	}

	oldConfigData := []byte("notes_dir: ~/recap\nai_provider: github_models\n")
	if err := os.WriteFile(oldConfigPath, oldConfigData, 0o600); err != nil {
		t.Fatalf("WriteFile(old config) error = %v", err)
	}

	message, err := MigrateOldConfig()
	if err != nil {
		t.Fatalf("MigrateOldConfig() error = %v", err)
	}

	wantMessage := "Migrated config from ~/.config/notes/ to ~/.config/recap/"
	if message != wantMessage {
		t.Fatalf("MigrateOldConfig() message = %q, want %q", message, wantMessage)
	}

	newConfigPath := filepath.Join(userConfigDir, "recap", "config.yaml")
	newConfigData, err := os.ReadFile(newConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(new config) error = %v", err)
	}

	if string(newConfigData) != string(oldConfigData) {
		t.Fatalf("migrated config content mismatch: got %q, want %q", string(newConfigData), string(oldConfigData))
	}
}

func TestDefaultConfigSaveAndLoadRoundTrip(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	input := &Config{
		NotesDir:         "~/recap-work",
		AIProvider:       "openrouter",
		GitHubModel:      "gpt-4.1",
		OpenRouterModel:  "openai/gpt-4o-mini",
		OpenRouterAPIKey: "token-value",
		LMStudioURL:      "",
		LMStudioModel:    "",
		EmailLanguage:    "en",
	}

	if err := Save(input, configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantNotesDir := filepath.Join(homeDir, "recap-work")
	if loaded.NotesDir != wantNotesDir {
		t.Fatalf("loaded.NotesDir = %q, want %q", loaded.NotesDir, wantNotesDir)
	}
	if loaded.AIProvider != "openrouter" {
		t.Fatalf("loaded.AIProvider = %q, want %q", loaded.AIProvider, "openrouter")
	}
	if loaded.GitHubModel != "gpt-4.1" {
		t.Fatalf("loaded.GitHubModel = %q, want %q", loaded.GitHubModel, "gpt-4.1")
	}
	if loaded.OpenRouterModel != "openai/gpt-4o-mini" {
		t.Fatalf("loaded.OpenRouterModel = %q, want %q", loaded.OpenRouterModel, "openai/gpt-4o-mini")
	}
	if loaded.OpenRouterAPIKey != "token-value" {
		t.Fatalf("loaded.OpenRouterAPIKey = %q, want %q", loaded.OpenRouterAPIKey, "token-value")
	}
	if loaded.LMStudioURL != "http://localhost:1234/v1" {
		t.Fatalf("loaded.LMStudioURL = %q, want %q", loaded.LMStudioURL, "http://localhost:1234/v1")
	}
	if loaded.EmailLanguage != "en" {
		t.Fatalf("loaded.EmailLanguage = %q, want %q", loaded.EmailLanguage, "en")
	}
}

func TestDefaultConfigLoadCreatesConfigWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configPath := filepath.Join(t.TempDir(), "missing-config.yaml")

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if filepath.Base(loaded.NotesDir) != "recap" {
		t.Fatalf("Load() default NotesDir = %q, want path ending with %q", loaded.NotesDir, "recap")
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created at %q, stat error = %v", configPath, err)
	}
}
