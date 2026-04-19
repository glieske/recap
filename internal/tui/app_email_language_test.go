package tui

import (
	"testing"

	"github.com/glieske/recap/internal/config"
)

func TestAppEmailLanguageHelper(t *testing.T) {
	t.Run("returns first configured language", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"pl", "en", "de"}}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "pl" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "pl")
		}
	})

	t.Run("returns first configured language from list", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"en", "pl", "de"}}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("nil config defaults to en", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("empty EmailLanguages defaults to en", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{}}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("returns first of multi-language list", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"de", "fr"}}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "de" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "de")
		}
	})
}
