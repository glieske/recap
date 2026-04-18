package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func updateEditorFocusModelForTest(t *testing.T, m EditorModel, msg tea.Msg) (EditorModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func setupSplitEditorForFocusTests(t *testing.T) EditorModel {
	t.Helper()

	m := NewEditorModel(nil, nil, 120, 24, "", "")
	m.textarea.SetValue("left")
	sm := NewSummaryModel("right", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if !updated.splitMode {
		t.Fatalf("expected split mode enabled during test setup")
	}

	return updated
}

func TestFocusTabSwitchesFromLeftToRightAndBack(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)

	if m.focusedPane != 0 {
		t.Fatalf("expected focusedPane=0 at split entry, got %d", m.focusedPane)
	}
	if !m.textarea.Focused() {
		t.Fatalf("expected textarea focused on split entry")
	}
	if m.summaryModel.textarea.Focused() {
		t.Fatalf("expected summary blurred on split entry")
	}

	toRight, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("expected nil cmd for split tab focus switch")
	}
	if toRight.focusedPane != 1 {
		t.Fatalf("expected focusedPane=1 after first tab, got %d", toRight.focusedPane)
	}
	if toRight.textarea.Focused() {
		t.Fatalf("expected textarea blurred after first tab")
	}
	if !toRight.summaryModel.textarea.Focused() {
		t.Fatalf("expected summary focused after first tab")
	}

	toLeft, cmd := updateEditorFocusModelForTest(t, toRight, tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("expected nil cmd for split tab focus switch back")
	}
	if toLeft.focusedPane != 0 {
		t.Fatalf("expected focusedPane=0 after second tab, got %d", toLeft.focusedPane)
	}
	if !toLeft.textarea.Focused() {
		t.Fatalf("expected textarea focused after second tab")
	}
	if toLeft.summaryModel.textarea.Focused() {
		t.Fatalf("expected summary blurred after second tab")
	}
}

func TestFocusTabOutsideSplitPassesToTextareaAndMarksDirty(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if updated.splitMode {
		t.Fatalf("expected splitMode=false for non-split tab")
	}
	if !updated.dirty {
		t.Fatalf("expected dirty=true when tab is routed to textarea")
	}
}

func TestFocusEscInSplitCollapsesAndRestoresHeight(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)

	collapsed, cmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatalf("expected nil cmd when collapsing split on esc")
	}
	if collapsed.splitMode {
		t.Fatalf("expected splitMode=false after esc collapse")
	}
	if collapsed.focusedPane != 0 {
		t.Fatalf("expected focusedPane reset to 0 after esc collapse, got %d", collapsed.focusedPane)
	}
	if !collapsed.textarea.Focused() {
		t.Fatalf("expected textarea focused after esc collapse")
	}
	if collapsed.summaryModel.textarea.Focused() {
		t.Fatalf("expected summary blurred after esc collapse")
	}

	fullHeight := maxEditorHeight(collapsed.height)
	lineCount := strings.Count(collapsed.textarea.View(), "\n") + 1
	if lineCount != fullHeight {
		t.Fatalf("expected textarea view lines %d after collapse, got %d", fullHeight, lineCount)
	}
}

func TestFocusEscOutsideSplitPassesToTextarea(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if updated.splitMode {
		t.Fatalf("expected splitMode to remain false on esc outside split")
	}
}

func TestFocusGlobalHotkeysStillHandledWhenSummaryPaneFocused(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusedPane != 1 {
		t.Fatalf("expected focusedPane=1 after tab, got %d", m.focusedPane)
	}

	beforeSummary := m.summaryModel.Value()
	beforeTextarea := m.textarea.Value()

	ctrlAUpdated, ctrlACmd := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if ctrlACmd == nil {
		t.Fatalf("expected cmd for ctrl+a")
	}
	if _, ok := ctrlACmd().(TriggerAIMsg); !ok {
		t.Fatalf("expected TriggerAIMsg from ctrl+a")
	}
	if ctrlAUpdated.summaryModel.Value() != beforeSummary {
		t.Fatalf("expected summary unchanged for ctrl+a, got %q", ctrlAUpdated.summaryModel.Value())
	}

	ctrlEUpdated, ctrlECmd := updateEditorFocusModelForTest(t, ctrlAUpdated, tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if ctrlECmd == nil {
		t.Fatalf("expected cmd for ctrl+e")
	}
	if _, ok := ctrlECmd().(TriggerEmailMsg); !ok {
		t.Fatalf("expected TriggerEmailMsg from ctrl+e")
	}
	if ctrlEUpdated.summaryModel.Value() != beforeSummary {
		t.Fatalf("expected summary unchanged for ctrl+e, got %q", ctrlEUpdated.summaryModel.Value())
	}

	ctrlTUpdated, ctrlTCmd := updateEditorFocusModelForTest(t, ctrlEUpdated, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if ctrlTCmd != nil {
		t.Fatalf("expected nil cmd for ctrl+t")
	}
	if ctrlTUpdated.summaryModel.Value() != beforeSummary {
		t.Fatalf("expected summary unchanged for ctrl+t, got %q", ctrlTUpdated.summaryModel.Value())
	}
	if ctrlTUpdated.textarea.Value() == beforeTextarea {
		t.Fatalf("expected ctrl+t to modify textarea even when summary pane focused")
	}
	if !ctrlTUpdated.dirty {
		t.Fatalf("expected ctrl+t to set dirty=true")
	}
}

func TestFocusRegularKeysRouteToSummaryWhenSummaryPaneFocused(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	m, _ = updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusedPane != 1 {
		t.Fatalf("expected focusedPane=1 after tab, got %d", m.focusedPane)
	}

	beforeSummary := m.summaryModel.Value()
	beforeTextarea := m.textarea.Value()

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Text: "x"})
	if updated.summaryModel.Value() != beforeSummary+"x" {
		t.Fatalf("expected summary value %q, got %q", beforeSummary+"x", updated.summaryModel.Value())
	}
	if updated.textarea.Value() != beforeTextarea {
		t.Fatalf("expected textarea to remain %q, got %q", beforeTextarea, updated.textarea.Value())
	}
}

func TestFocusViewSplitModeUsesDifferentBordersPerPane(t *testing.T) {
	m := setupSplitEditorForFocusTests(t)
	view := m.View().Content

	if !strings.Contains(view, "╭") {
		t.Fatalf("expected rounded border rune for focused pane, got %q", view)
	}
	if !strings.Contains(view, "┌") {
		t.Fatalf("expected normal border rune for unfocused pane, got %q", view)
	}
}

func TestFocusEnteringSplitAdjustsInnerPaneHeight(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")
	sm := NewSummaryModel("content", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := updateEditorFocusModelForTest(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	expectedInnerHeight := maxEditorHeight(updated.height) - 2
	if expectedInnerHeight < 1 {
		expectedInnerHeight = 1
	}

	if updated.summaryModel.height != expectedInnerHeight {
		t.Fatalf("expected summary model height %d, got %d", expectedInnerHeight, updated.summaryModel.height)
	}

	lineCount := strings.Count(updated.textarea.View(), "\n") + 1
	if lineCount != expectedInnerHeight {
		t.Fatalf("expected textarea view lines %d in split mode, got %d", expectedInnerHeight, lineCount)
	}
}
