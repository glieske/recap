package tui

import (
	"testing"

	"github.com/glieske/recap/internal/config"
)

func TestAppEmailLanguageHelper(t *testing.T) {
	t.Run("returns configured language when non-empty", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguage: "en"}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("nil config defaults to pl", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "pl" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "pl")
		}
	})

	t.Run("empty config email language defaults to pl", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguage: ""}, nil, nil, "", false, "")

		if got := m.emailLanguage(); got != "pl" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "pl")
		}
	})
}
