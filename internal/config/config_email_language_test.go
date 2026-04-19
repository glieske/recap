package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigEmailLanguageDefaultsAndNormalization(t *testing.T) {
	t.Run("default config uses default enabled codes", func(t *testing.T) {
		cfg, err := defaultConfig()
		if err != nil {
			t.Fatalf("defaultConfig() error = %v", err)
		}

		want := []string{"en", "pl", "de", "no"}
		if len(cfg.EmailLanguages) != len(want) {
			t.Fatalf("EmailLanguages = %v, want %v", cfg.EmailLanguages, want)
		}
		for i, code := range want {
			if cfg.EmailLanguages[i] != code {
				t.Fatalf("EmailLanguages[%d] = %q, want %q", i, cfg.EmailLanguages[i], code)
			}
		}
	})

	t.Run("empty config defaults to default enabled codes", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		want := []string{"en", "pl", "de", "no"}
		if len(cfg.EmailLanguages) != len(want) {
			t.Fatalf("EmailLanguages = %v, want %v", cfg.EmailLanguages, want)
		}
	})

	t.Run("invalid codes are filtered out", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_languages:",
			"  - en",
			"  - xx",
			"  - pl",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		want := []string{"en", "pl"}
		if len(cfg.EmailLanguages) != len(want) {
			t.Fatalf("EmailLanguages = %v, want %v", cfg.EmailLanguages, want)
		}
	})

	t.Run("all invalid codes fallback to defaults", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_languages:",
			"  - xx",
			"  - yy",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		want := []string{"en", "pl", "de", "no"}
		if len(cfg.EmailLanguages) != len(want) {
			t.Fatalf("EmailLanguages = %v, want %v", cfg.EmailLanguages, want)
		}
	})

	t.Run("max 5 languages enforced by truncation", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_languages:",
			"  - en",
			"  - pl",
			"  - de",
			"  - no",
			"  - zh",
			"  - hi",
			"  - es",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		if len(cfg.EmailLanguages) != 5 {
			t.Fatalf("EmailLanguages length = %d, want 5", len(cfg.EmailLanguages))
		}
	})

	t.Run("valid languages are preserved in order", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_languages:",
			"  - ja",
			"  - fr",
			"  - de",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		want := []string{"ja", "fr", "de"}
		if len(cfg.EmailLanguages) != len(want) {
			t.Fatalf("EmailLanguages = %v, want %v", cfg.EmailLanguages, want)
		}
		for i, code := range want {
			if cfg.EmailLanguages[i] != code {
				t.Fatalf("EmailLanguages[%d] = %q, want %q", i, cfg.EmailLanguages[i], code)
			}
		}
	})

}
