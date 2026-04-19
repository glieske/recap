package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigEmailLanguageDefaultsAndNormalization(t *testing.T) {
	t.Run("default config uses en", func(t *testing.T) {
		cfg, err := defaultConfig()
		if err != nil {
			t.Fatalf("defaultConfig() error = %v", err)
		}

		if cfg.EmailLanguage != "en" {
			t.Fatalf("EmailLanguage = %q, want %q", cfg.EmailLanguage, "en")
		}
	})

	t.Run("empty email_language normalizes to en", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_language: \"\"",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		if cfg.EmailLanguage != "en" {
			t.Fatalf("EmailLanguage = %q, want %q", cfg.EmailLanguage, "en")
		}
	})

	t.Run("invalid email_language normalizes to en", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		content := []byte(strings.Join([]string{
			"notes_dir: /tmp/notes",
			"email_language: fr",
			"",
		}, "\n"))

		if err := os.WriteFile(configPath, content, 0o600); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load(%q) error = %v", configPath, err)
		}

		if cfg.EmailLanguage != "en" {
			t.Fatalf("EmailLanguage = %q, want %q", cfg.EmailLanguage, "en")
		}
	})

	t.Run("valid values en pl no are preserved", func(t *testing.T) {
		tests := []struct {
			name     string
			language string
		}{
			{name: "en preserved", language: "en"},
			{name: "pl preserved", language: "pl"},
			{name: "no preserved", language: "no"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				configPath := filepath.Join(t.TempDir(), "config.yaml")
				content := []byte(strings.Join([]string{
					"notes_dir: /tmp/notes",
					"email_language: " + tt.language,
					"",
				}, "\n"))

				if err := os.WriteFile(configPath, content, 0o600); err != nil {
					t.Fatalf("WriteFile(config) error = %v", err)
				}

				cfg, err := Load(configPath)
				if err != nil {
					t.Fatalf("Load(%q) error = %v", configPath, err)
				}

				if cfg.EmailLanguage != tt.language {
					t.Fatalf("EmailLanguage = %q, want %q", cfg.EmailLanguage, tt.language)
				}
			})
		}
	})
}
