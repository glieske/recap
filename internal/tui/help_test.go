package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewHelpModelReturnsZeroValueModel(t *testing.T) {
	m := NewHelpModel()

	if m.width != 0 {
		t.Fatalf("expected width to be 0, got %d", m.width)
	}
	if m.height != 0 {
		t.Fatalf("expected height to be 0, got %d", m.height)
	}
}

func TestHelpModelInitReturnsNil(t *testing.T) {
	m := NewHelpModel()

	cmd := m.Init()
	if cmd != nil {
		t.Fatalf("expected Init() to return nil cmd")
	}
}

func TestHelpModelUpdateWindowSizeSetsDimensions(t *testing.T) {
	m := NewHelpModel()

	updatedModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatalf("expected nil cmd, got non-nil")
	}

	updated, ok := updatedModel.(HelpModel)
	if !ok {
		t.Fatalf("expected updated model type HelpModel")
	}

	if updated.width != 120 {
		t.Fatalf("expected width 120, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Fatalf("expected height 40, got %d", updated.height)
	}
}

func TestHelpModelUpdateUnknownMessagePassesThrough(t *testing.T) {
	original := HelpModel{width: 77, height: 22}

	updatedModel, cmd := original.Update("unknown-message")
	if cmd != nil {
		t.Fatalf("expected nil cmd, got non-nil")
	}

	updated, ok := updatedModel.(HelpModel)
	if !ok {
		t.Fatalf("expected updated model type HelpModel")
	}

	if updated.width != original.width {
		t.Fatalf("expected width to remain %d, got %d", original.width, updated.width)
	}
	if updated.height != original.height {
		t.Fatalf("expected height to remain %d, got %d", original.height, updated.height)
	}
}

func TestHelpModelViewContainsSectionHeadersAndBindingsAndTitle(t *testing.T) {
	// Test each screen separately — help is now context-sensitive.
	t.Run("editor screen shows editor and split pane sections", func(t *testing.T) {
		m := HelpModel{screen: ScreenEditor}
		view := m.View().Content
		for _, sub := range []string{"Global", "Editor", "Split Pane", "Ctrl+S", "Ctrl+A", "Ctrl+E", "Ctrl+P", "Keybindings"} {
			if !strings.Contains(view, sub) {
				t.Fatalf("expected view to contain %q", sub)
			}
		}
	})
	t.Run("meeting list screen shows meeting list section", func(t *testing.T) {
		m := HelpModel{screen: ScreenMeetingList}
		view := m.View().Content
		for _, sub := range []string{"Global", "Meeting List", "Enter", "n", "f", "t", "d", "Keybindings"} {
			if !strings.Contains(view, sub) {
				t.Fatalf("expected view to contain %q", sub)
			}
		}
	})
	t.Run("email screen shows email section", func(t *testing.T) {
		m := HelpModel{screen: ScreenEmail}
		view := m.View().Content
		for _, sub := range []string{"Global", "Email", "q / Ctrl+C", "Keybindings"} {
			if !strings.Contains(view, sub) {
				t.Fatalf("expected view to contain %q", sub)
			}
		}
	})
}

func TestHelpModelViewZeroDimensionsReturnsNonEmptyUncenteredBox(t *testing.T) {
	m := NewHelpModel()
	view := m.View().Content

	if view == "" {
		t.Fatalf("expected non-empty view when dimensions are zero")
	}
	if !strings.Contains(view, "Keybindings") {
		t.Fatalf("expected view to contain title text")
	}
}

func TestHelpModelViewWithDimensionsReturnsNonEmptyCenteredOutput(t *testing.T) {
	m := HelpModel{width: 100, height: 30}
	view := m.View().Content

	if view == "" {
		t.Fatalf("expected non-empty centered view")
	}
	if !strings.Contains(view, "Keybindings") {
		t.Fatalf("expected centered view to contain title text")
	}
}
