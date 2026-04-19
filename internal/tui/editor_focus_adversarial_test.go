package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFocusAdversarialRapidTabTogglingAlternatesWithoutStateCorruption(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)

	for i := 1; i <= 20; i++ {
		updated, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
		if cmd != nil {
			t.Fatalf("expected nil cmd on split tab toggle #%d", i)
		}

		expectedPane := i % 2
		if updated.focusedPane != expectedPane {
			t.Fatalf("toggle #%d: expected focusedPane=%d, got %d", i, expectedPane, updated.focusedPane)
		}

		if expectedPane == 0 {
			if !updated.textarea.Focused() {
				t.Fatalf("toggle #%d: expected textarea focused when focusedPane=0", i)
			}
			if updated.summaryModel.textarea.Focused() {
				t.Fatalf("toggle #%d: expected summary blurred when focusedPane=0", i)
			}
		} else {
			if updated.textarea.Focused() {
				t.Fatalf("toggle #%d: expected textarea blurred when focusedPane=1", i)
			}
			if !updated.summaryModel.textarea.Focused() {
				t.Fatalf("toggle #%d: expected summary focused when focusedPane=1", i)
			}
		}

		m = updated
	}
}

func TestFocusAdversarialTabWhenSplitWithoutSummaryRoutesToTextarea(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	m.splitMode = true
	m.hasSummaryModel = false
	m.focusedPane = 0

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})

	if !updated.splitMode {
		t.Fatalf("expected splitMode=true to remain unchanged")
	}
	if updated.hasSummaryModel {
		t.Fatalf("expected hasSummaryModel=false to remain unchanged")
	}
	if updated.focusedPane != 0 {
		t.Fatalf("expected focusedPane=0, got %d", updated.focusedPane)
	}
	if !updated.dirty {
		t.Fatalf("expected dirty=true when tab is routed to textarea in edge split state")
	}
}

func TestFocusAdversarialEscFromRightPaneCollapsesSplitAndResetsFocus(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusedPane != 1 {
		t.Fatalf("precondition failed: expected focusedPane=1 before esc, got %d", m.focusedPane)
	}

	updated, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatalf("expected nil cmd when collapsing split via esc")
	}
	if updated.splitMode {
		t.Fatalf("expected splitMode=false after esc")
	}
	if updated.focusedPane != 0 {
		t.Fatalf("expected focusedPane reset to 0 after esc, got %d", updated.focusedPane)
	}
	if !updated.textarea.Focused() {
		t.Fatalf("expected textarea focused after esc collapse")
	}
	if updated.summaryModel.textarea.Focused() {
		t.Fatalf("expected summary blurred after esc collapse")
	}
}

func TestFocusAdversarialCtrlAOnRightPaneEmitsTriggerAIOnly(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusedPane != 1 {
		t.Fatalf("precondition failed: expected focusedPane=1, got %d", m.focusedPane)
	}

	beforeSummary := m.summaryModel.Value()
	updated, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})

	if cmd == nil {
		t.Fatalf("expected non-nil cmd for ctrl+a")
	}
	msg := cmd()
	if _, ok := msg.(TriggerAIMsg); !ok {
		t.Fatalf("expected TriggerAIMsg, got %T", msg)
	}
	if updated.summaryModel.Value() != beforeSummary {
		t.Fatalf("expected summary unchanged on ctrl+a, got %q", updated.summaryModel.Value())
	}
}

func TestFocusAdversarialCtrlEOnRightPaneEmitsTriggerEmailOnly(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusedPane != 1 {
		t.Fatalf("precondition failed: expected focusedPane=1, got %d", m.focusedPane)
	}

	beforeSummary := m.summaryModel.Value()
	updated, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})

	if cmd == nil {
		t.Fatalf("expected non-nil cmd for ctrl+e")
	}
	msg := cmd()
	if _, ok := msg.(TriggerEmailMsg); !ok {
		t.Fatalf("expected TriggerEmailMsg, got %T", msg)
	}
	if updated.summaryModel.Value() != beforeSummary {
		t.Fatalf("expected summary unchanged on ctrl+e, got %q", updated.summaryModel.Value())
	}
}

func TestFocusAdversarialWindowSizeHeightFiveClampsInnerPaneHeightToOne(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	sm := NewSummaryModel("test", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if !m.splitMode {
		t.Fatalf("precondition failed: expected splitMode=true")
	}

	updated, _ := updateEditorFocusModelForTest(t, m, tea.WindowSizeMsg{Width: 120, Height: 5})

	if updated.summaryModel.height != 1 {
		t.Fatalf("expected summary model height=1 after clamp, got %d", updated.summaryModel.height)
	}
	lineCount := strings.Count(updated.textarea.View(), "\n") + 1
	if lineCount != 1 {
		t.Fatalf("expected textarea rendered line count=1 after clamp, got %d", lineCount)
	}
}

func TestFocusAdversarialResizeNarrowKeepsSplitActive(t *testing.T) {
	m := NewEditorModel(nil, nil, 100, 24, "", "", "")
	sm := NewSummaryModel("test", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if !m.splitMode {
		t.Fatalf("precondition failed: expected splitMode=true at width=100")
	}

	updated, _ := updateEditorFocusModelForTest(t, m, tea.WindowSizeMsg{Width: 50, Height: 24})

	if !updated.splitMode {
		t.Fatalf("expected splitMode=true after resize to width=50")
	}
	if updated.width != 50 {
		t.Fatalf("expected width=50 after resize, got %d", updated.width)
	}
	view := updated.View().Content
	if !strings.Contains(view, "│") {
		t.Fatalf("expected split divider in view after narrow resize")
	}
}
