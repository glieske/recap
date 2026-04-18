package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfigPathReturnsNonEmptyRecapConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	if path == "" {
		t.Fatalf("DefaultConfigPath() returned empty path")
	}

	wantSuffix := filepath.Join("recap", "config.yaml")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Fatalf("DefaultConfigPath() = %q, want suffix %q", path, wantSuffix)
	}
}

func TestLoadEmptyPathCreatesDefaultConfigAndFile(t *testing.T) {
	tempRoot := t.TempDir()
	homeDir := filepath.Join(tempRoot, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) error = %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempRoot, "xdg-config"))
	t.Setenv("APPDATA", filepath.Join(tempRoot, "appdata"))

	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	_, statErr := os.Stat(configPath)
	if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected config file to not exist before Load, stat error = %v", statErr)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}

	if cfg.AIProvider != defaultAIProvider {
		t.Fatalf("AIProvider = %q, want %q", cfg.AIProvider, defaultAIProvider)
	}

	if cfg.GitHubModel != defaultGitHubModel {
		t.Fatalf("GitHubModel = %q, want %q", cfg.GitHubModel, defaultGitHubModel)
	}

	if cfg.OpenRouterModel != defaultOpenRouterModel {
		t.Fatalf("OpenRouterModel = %q, want %q", cfg.OpenRouterModel, defaultOpenRouterModel)
	}

	if filepath.Base(cfg.NotesDir) != "recap" {
		t.Fatalf("NotesDir = %q, want path ending with %q", cfg.NotesDir, "recap")
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created at %q, stat error = %v", configPath, err)
	}
}

func TestLoadExistingConfigReadsValues(t *testing.T) {
	tempRoot := t.TempDir()
	homeDir := filepath.Join(tempRoot, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) error = %v", err)
	}
	t.Setenv("HOME", homeDir)

	configPath := filepath.Join(tempRoot, "config", "custom.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(config dir) error = %v", err)
	}

	content := []byte(strings.Join([]string{
		"notes_dir: /tmp/my-notes",
		"ai_provider: openrouter",
		"github_model: gpt-custom",
		"openrouter_model: anthropic/claude-3.5-sonnet",
		"openrouter_api_key: sk-test-key",
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", configPath, err)
	}

	if cfg.NotesDir != "/tmp/my-notes" {
		t.Fatalf("NotesDir = %q, want %q", cfg.NotesDir, "/tmp/my-notes")
	}
	if cfg.AIProvider != "openrouter" {
		t.Fatalf("AIProvider = %q, want %q", cfg.AIProvider, "openrouter")
	}
	if cfg.GitHubModel != "gpt-custom" {
		t.Fatalf("GitHubModel = %q, want %q", cfg.GitHubModel, "gpt-custom")
	}
	if cfg.OpenRouterModel != "anthropic/claude-3.5-sonnet" {
		t.Fatalf("OpenRouterModel = %q, want %q", cfg.OpenRouterModel, "anthropic/claude-3.5-sonnet")
	}
	if cfg.OpenRouterAPIKey != "sk-test-key" {
		t.Fatalf("OpenRouterAPIKey = %q, want %q", cfg.OpenRouterAPIKey, "sk-test-key")
	}
}

func TestSaveCreatesParentDirsAndWritesValidYAML(t *testing.T) {
	tempRoot := t.TempDir()
	targetPath := filepath.Join(tempRoot, "nested", "notes", "config.yaml")

	cfg := &Config{
		NotesDir:         filepath.Join(tempRoot, "notes"),
		AIProvider:       "openrouter",
		GitHubModel:      "gpt-save-test",
		OpenRouterModel:  "openrouter/model-test",
		OpenRouterAPIKey: "secret-token",
	}

	if err := Save(cfg, targetPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(filepath.Dir(targetPath)); err != nil {
		t.Fatalf("expected parent directory to be created, stat error = %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", targetPath, err)
	}

	var decoded Config
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("saved YAML is invalid, unmarshal error = %v", err)
	}

	if decoded.NotesDir != cfg.NotesDir {
		t.Fatalf("saved NotesDir = %q, want %q", decoded.NotesDir, cfg.NotesDir)
	}
	if decoded.AIProvider != cfg.AIProvider {
		t.Fatalf("saved AIProvider = %q, want %q", decoded.AIProvider, cfg.AIProvider)
	}
	if decoded.GitHubModel != cfg.GitHubModel {
		t.Fatalf("saved GitHubModel = %q, want %q", decoded.GitHubModel, cfg.GitHubModel)
	}
	if decoded.OpenRouterModel != cfg.OpenRouterModel {
		t.Fatalf("saved OpenRouterModel = %q, want %q", decoded.OpenRouterModel, cfg.OpenRouterModel)
	}
	if decoded.OpenRouterAPIKey != cfg.OpenRouterAPIKey {
		t.Fatalf("saved OpenRouterAPIKey = %q, want %q", decoded.OpenRouterAPIKey, cfg.OpenRouterAPIKey)
	}
}

func TestLoadExpandsHomePathVariants(t *testing.T) {
	tempRoot := t.TempDir()
	homeDir := filepath.Join(tempRoot, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) error = %v", err)
	}
	t.Setenv("HOME", homeDir)

	tests := []struct {
		name     string
		notesDir string
		want     string
	}{
		{name: "tilde only", notesDir: "~", want: homeDir},
		{name: "tilde slash", notesDir: "~/docs", want: filepath.Join(homeDir, "docs")},
		{name: "tilde backslash", notesDir: "~\\docs", want: filepath.Join(homeDir, "docs")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			content := []byte(strings.Join([]string{
				"notes_dir: '" + tt.notesDir + "'",
				"ai_provider: github_models",
				"",
			}, "\n"))

			if err := os.WriteFile(configPath, content, 0o600); err != nil {
				t.Fatalf("WriteFile(config) error = %v", err)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load(%q) error = %v", configPath, err)
			}

			if cfg.NotesDir != tt.want {
				t.Fatalf("NotesDir = %q, want %q", cfg.NotesDir, tt.want)
			}
		})
	}
}

func TestLoadInvalidAIProviderResetsToDefault(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(strings.Join([]string{
		"notes_dir: /tmp/notes",
		"ai_provider: invalid-provider",
		"github_model: gpt-any",
		"openrouter_model: any/model",
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", configPath, err)
	}

	if cfg.AIProvider != defaultAIProvider {
		t.Fatalf("AIProvider = %q, want %q", cfg.AIProvider, defaultAIProvider)
	}
}

func TestLoadEmptyPathCreatesDefaultLMStudioConfig(t *testing.T) {
	tempRoot := t.TempDir()
	homeDir := filepath.Join(tempRoot, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) error = %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempRoot, "xdg-config"))
	t.Setenv("APPDATA", filepath.Join(tempRoot, "appdata"))

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}

	if cfg.LMStudioURL != defaultLMStudioURL {
		t.Fatalf("LMStudioURL = %q, want %q", cfg.LMStudioURL, defaultLMStudioURL)
	}

	if cfg.LMStudioModel != "" {
		t.Fatalf("LMStudioModel = %q, want empty string", cfg.LMStudioModel)
	}
}

func TestLoadKeepsLMStudioProvider(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(strings.Join([]string{
		"notes_dir: /tmp/notes",
		"ai_provider: lm_studio",
		"lm_studio_url: http://localhost:1234/v1",
		"lm_studio_model: local-model",
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", configPath, err)
	}

	if cfg.AIProvider != "lm_studio" {
		t.Fatalf("AIProvider = %q, want %q", cfg.AIProvider, "lm_studio")
	}
}

func TestLoadKeepsCustomLMStudioURL(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	const customURL = "http://127.0.0.1:8888/v1"
	content := []byte(strings.Join([]string{
		"notes_dir: /tmp/notes",
		"ai_provider: lm_studio",
		"lm_studio_url: " + customURL,
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", configPath, err)
	}

	if cfg.LMStudioURL != customURL {
		t.Fatalf("LMStudioURL = %q, want %q", cfg.LMStudioURL, customURL)
	}
}

func TestLoadEmptyLMStudioURLDefaultsToLocalhost(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(strings.Join([]string{
		"notes_dir: /tmp/notes",
		"ai_provider: lm_studio",
		"lm_studio_url: \"\"",
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", configPath, err)
	}

	if cfg.LMStudioURL != defaultLMStudioURL {
		t.Fatalf("LMStudioURL = %q, want %q", cfg.LMStudioURL, defaultLMStudioURL)
	}
}
