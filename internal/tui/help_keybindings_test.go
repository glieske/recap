package tui

import (
	"strings"
	"testing"
)

func TestHelpKeybindings(t *testing.T) {
	model := NewHelpModel()
	view := model.View().Content

	t.Run("editor section contains new AI and split bindings", func(t *testing.T) {
		checks := []string{
			"Editor",
			"Toggle split view",
			"Select AI provider",
			"Ctrl+G",
		}

		for _, want := range checks {
			if !strings.Contains(view, want) {
				t.Fatalf("expected help view to contain %q", want)
			}
		}
	})

	t.Run("split pane section contains documented controls", func(t *testing.T) {
		checks := []string{
			"Split Pane",
			"Tab",
			"Switch focus between panes",
			"Ctrl+S",
			"Esc",
			"Collapse split view",
		}

		for _, want := range checks {
			if !strings.Contains(view, want) {
				t.Fatalf("expected help view to contain %q", want)
			}
		}
	})

	t.Run("email section contains language cycle binding", func(t *testing.T) {
		checks := []string{
			"Email",
			"Cycle email language",
			"l",
		}

		for _, want := range checks {
			if !strings.Contains(view, want) {
				t.Fatalf("expected help view to contain %q", want)
			}
		}
	})
}
