package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestSplitSetGetSummaryModelAndSplitFlag(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	if m.IsSplitMode() {
		t.Fatalf("expected split mode false by default")
	}

	sm := NewSummaryModel("summary", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	got := m.GetSummaryModel()
	if got.Value() != "summary" {
		t.Fatalf("expected summary content %q, got %q", "summary", got.Value())
	}
}

func TestSplitCtrlPWithoutSummaryEmitsTogglePreviewMsg(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updatedModel, ok := updated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updated)
	}
	if updatedModel.IsSplitMode() {
		t.Fatalf("expected split mode to remain false without summary model")
	}
	if cmd == nil {
		t.Fatalf("expected non-nil command for ctrl+p fallback")
	}

	msg := cmd()
	if _, ok := msg.(TogglePreviewMsg); !ok {
		t.Fatalf("expected TogglePreviewMsg, got %T", msg)
	}
}

func TestSplitCtrlPWithSummaryTooNarrowSetsStatusAndDoesNotToggle(t *testing.T) {
	m := NewEditorModel(nil, nil, 99, 24, "", "", "")
	sm := NewSummaryModel("summary", "", "", nil, 49, 20)
	m.SetSummaryModel(sm)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updatedModel, ok := updated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updated)
	}

	if cmd != nil {
		t.Fatalf("expected nil command when terminal is too narrow")
	}
	if updatedModel.IsSplitMode() {
		t.Fatalf("expected split mode false when width < 100")
	}
	if updatedModel.statusMsg != "Terminal too narrow for split view (need 100+ cols)" {
		t.Fatalf("expected narrow terminal status message, got %q", updatedModel.statusMsg)
	}
	if time.Until(updatedModel.statusExpiry) <= 0 {
		t.Fatalf("expected future status expiry for narrow terminal warning")
	}
}

func TestSplitCtrlPWithSummaryWideTogglesOnAndSetsPaneWidths(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	sm := NewSummaryModel("summary", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updatedModel, ok := updated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updated)
	}

	if cmd != nil {
		t.Fatalf("expected nil command when toggling split mode")
	}
	if !updatedModel.IsSplitMode() {
		t.Fatalf("expected split mode true after ctrl+p with width >= 100")
	}
	if updatedModel.focusedPane != 0 {
		t.Fatalf("expected focusedPane=0, got %d", updatedModel.focusedPane)
	}

	leftWidth := (updatedModel.width - 1) / 2
	rightWidth := updatedModel.width - leftWidth - 1
	rightContentWidth := rightWidth - 2
	if updatedModel.summaryModel.width != rightContentWidth {
		t.Fatalf("expected summary width %d, got %d", rightContentWidth, updatedModel.summaryModel.width)
	}
}

func TestSplitCtrlPToggleOffRestoresFullTextareaWidth(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	sm := NewSummaryModel("summary", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	firstUpdated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	firstModel := firstUpdated.(EditorModel)
	if !firstModel.IsSplitMode() {
		t.Fatalf("expected split mode true after first ctrl+p")
	}

	secondUpdated, cmd := firstModel.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	secondModel, ok := secondUpdated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from second update, got %T", secondUpdated)
	}

	if cmd != nil {
		t.Fatalf("expected nil command when toggling split mode off")
	}
	if secondModel.IsSplitMode() {
		t.Fatalf("expected split mode false after second ctrl+p")
	}
}

func TestSplitViewInSplitModeContainsVerticalDivider(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	m.textarea.SetValue("left pane text")
	sm := NewSummaryModel("right pane text", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updatedModel := updated.(EditorModel)
	view := updatedModel.View().Content

	if !strings.Contains(view, "│") {
		t.Fatalf("expected split view to contain vertical divider, got %q", view)
	}
	if !strings.Contains(view, "left pane text") {
		t.Fatalf("expected split view to include left pane content, got %q", view)
	}
	if !strings.Contains(view, "right pane text") {
		t.Fatalf("expected split view to include right pane content, got %q", view)
	}
}

func TestSplitViewInSplitModeIncludesNotesHeader(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	m.textarea.SetValue("left pane text")
	sm := NewSummaryModel("right pane text", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	updatedModel := updated.(EditorModel)
	view := updatedModel.View().Content

	if !strings.Contains(view, "── Notes") {
		t.Fatalf("expected split view to include notes header %q, got %q", "── Notes", view)
	}
}

func TestSplitViewWithoutSplitModeShowsNormalEditorLayout(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	m.textarea.SetValue("normal editor body")

	view := m.View().Content

	if !strings.Contains(view, "normal editor body") {
		t.Fatalf("expected non-split view to include textarea body, got %q", view)
	}
	if !strings.Contains(view, "ctrl+s Save") {
		t.Fatalf("expected non-split view to include legend, got %q", view)
	}
	if strings.Contains(view, "│") {
		t.Fatalf("expected non-split view without vertical divider, got %q", view)
	}
}

func TestSplitWindowSizeMsgRecalculatesPaneWidthsInSplitMode(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	sm := NewSummaryModel("summary", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	splitModel := updated.(EditorModel)
	if !splitModel.IsSplitMode() {
		t.Fatalf("expected split mode enabled before resize")
	}

	resized, cmd := splitModel.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	resizedModel, ok := resized.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from resize update, got %T", resized)
	}

	if cmd != nil {
		t.Fatalf("expected nil command for window resize")
	}
	if resizedModel.width != 140 || resizedModel.height != 30 {
		t.Fatalf("expected model size 140x30, got %dx%d", resizedModel.width, resizedModel.height)
	}

	leftWidth := (140 - 1) / 2
	rightWidth := 140 - leftWidth - 1
	rightContentWidth := rightWidth - 2
	if resizedModel.summaryModel.width != rightContentWidth {
		t.Fatalf("expected resized right width %d, got %d", rightContentWidth, resizedModel.summaryModel.width)
	}
	innerPaneHeight := maxEditorHeight(30) - 2
	if resizedModel.summaryModel.height != innerPaneHeight {
		t.Fatalf("expected summary height %d, got %d", innerPaneHeight, resizedModel.summaryModel.height)
	}
}

func TestSplitRenderVerticalDividerHeightVariants(t *testing.T) {
	if got := renderVerticalDivider(0); got != "" {
		t.Fatalf("expected empty divider for height 0, got %q", got)
	}

	got := renderVerticalDivider(3)
	if got != "│\n│\n│" {
		t.Fatalf("expected exactly three divider lines, got %q", got)
	}
	if strings.Count(got, "│") != 3 {
		t.Fatalf("expected 3 divider rune occurrences, got %d", strings.Count(got, "│"))
	}
}

func TestSplitViewIncludesNotesHeaderInSplitMode(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "", "")
	sm := NewSummaryModel("right pane", "", "", nil, 60, 20)
	m.SetSummaryModel(sm)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	splitModel := updated.(EditorModel)

	view := splitModel.View().Content
	if !strings.Contains(view, "── Notes") {
		t.Fatalf("expected split view to contain notes header, got: %q", view)
	}
}
