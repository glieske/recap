package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSplitAdversarial(t *testing.T) {
	t.Run("rapid ctrl+p toggling 10x does not corrupt state", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 60, 20))

		for i := 0; i < 10; i++ {
			updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
			if cmd != nil {
				t.Fatalf("iteration %d: expected nil command for ctrl+p toggle, got non-nil", i)
			}

			typed, ok := updated.(EditorModel)
			if !ok {
				t.Fatalf("iteration %d: expected EditorModel, got %T", i, updated)
			}
			m = typed
		}

		if m.IsSplitMode() {
			t.Fatalf("expected split mode false after even number of toggles")
		}
		if !m.hasSummaryModel {
			t.Fatalf("expected summary model to remain configured")
		}
		if m.width != 120 || m.height != 24 {
			t.Fatalf("expected dimensions unchanged at 120x24, got %dx%d", m.width, m.height)
		}
	})

	t.Run("WindowSizeMsg width=0 in split mode does not panic", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 60, 20))

		on, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		split := on.(EditorModel)
		if !split.IsSplitMode() {
			t.Fatalf("expected split mode true before resize")
		}

		resized, cmd := split.Update(tea.WindowSizeMsg{Width: 0, Height: 24})
		if cmd != nil {
			t.Fatalf("expected nil command for resize")
		}
		updated := resized.(EditorModel)
		if !updated.IsSplitMode() {
			t.Fatalf("expected split mode to remain active after resize to width 0")
		}
		if updated.width != 0 {
			t.Fatalf("expected width 0 after resize, got %d", updated.width)
		}
	})

	t.Run("WindowSizeMsg width=1 in split mode does not panic", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 60, 20))

		on, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		split := on.(EditorModel)

		resized, _ := split.Update(tea.WindowSizeMsg{Width: 1, Height: 24})
		updated := resized.(EditorModel)
		if !updated.IsSplitMode() {
			t.Fatalf("expected split mode to remain active at width 1")
		}
		if updated.width != 1 {
			t.Fatalf("expected width 1 after resize, got %d", updated.width)
		}
	})

	t.Run("SetSummaryModel multiple times uses last model", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "", "")
		first := NewSummaryModel("first", "", "", nil, 60, 20)
		second := NewSummaryModel("second", "", "", nil, 60, 20)

		m.SetSummaryModel(first)
		m.SetSummaryModel(second)

		got := m.GetSummaryModel().Value()
		if got != "second" {
			t.Fatalf("expected last summary model content %q, got %q", "second", got)
		}

		updated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		typed := updated.(EditorModel)
		if typed.GetSummaryModel().Value() != "second" {
			t.Fatalf("expected split mode to use last injected summary model")
		}
	})

	t.Run("ctrl+p width exactly 99 shows narrow status", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 99, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 49, 20))

		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		if cmd != nil {
			t.Fatalf("expected nil command when width is 99")
		}
		typed := updated.(EditorModel)
		if typed.IsSplitMode() {
			t.Fatalf("expected split mode to remain false at width 99")
		}
		if typed.statusMsg != "Terminal too narrow for split view (need 100+ cols)" {
			t.Fatalf("expected specific narrow warning, got %q", typed.statusMsg)
		}
	})

	t.Run("ctrl+p width exactly 100 toggles split mode", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 100, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 50, 20))

		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		if cmd != nil {
			t.Fatalf("expected nil command when width is 100")
		}
		typed := updated.(EditorModel)
		if !typed.IsSplitMode() {
			t.Fatalf("expected split mode true at width 100")
		}
	})

	t.Run("renderVerticalDivider negative height returns empty", func(t *testing.T) {
		if got := renderVerticalDivider(-7); got != "" {
			t.Fatalf("expected empty string for negative height, got %q", got)
		}
	})

	t.Run("split remains active when resized below threshold", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "", "")
		m.SetSummaryModel(NewSummaryModel("summary", "", "", nil, 60, 20))

		on, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
		split := on.(EditorModel)
		if !split.IsSplitMode() {
			t.Fatalf("expected split mode true after toggle at width 120")
		}

		resized, _ := split.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		updated := resized.(EditorModel)
		if !updated.IsSplitMode() {
			t.Fatalf("expected split mode to remain true after shrinking below 100")
		}
		if updated.width != 80 {
			t.Fatalf("expected width 80 after resize, got %d", updated.width)
		}
	})
}
