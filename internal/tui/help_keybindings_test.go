package tui

import (
	"strings"
	"testing"
)

func TestHelpKeybindings(t *testing.T) {
	t.Run("editor section contains new AI and split bindings", func(t *testing.T) {
		model := NewHelpModel()
		model.screen = ScreenEditor
		view := model.View().Content
		checks := []string{
			"Editor",
			"Toggle split view",
			"Select AI provider",
			"Ctrl+A",
		}

		for _, want := range checks {
			if !strings.Contains(view, want) {
				t.Fatalf("expected help view to contain %q", want)
			}
		}
	})

	t.Run("split pane section contains documented controls", func(t *testing.T) {
		model := NewHelpModel()
		model.screen = ScreenEditor
		view := model.View().Content
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
		model := NewHelpModel()
		model.screen = ScreenEmail
		view := model.View().Content
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
