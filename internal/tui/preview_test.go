package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func updatePreviewModelForTest(t *testing.T, m PreviewModel, msg tea.Msg) (PreviewModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	previewModel, ok := updated.(PreviewModel)
	if !ok {
		t.Fatalf("expected PreviewModel from Update, got %T", updated)
	}

	return previewModel, cmd
}

func TestNewPreviewModelWithContent(t *testing.T) {
	m := NewPreviewModel("# Structured", "Weekly Sync", 80, 20)

	if m.content != "# Structured" {
		t.Fatalf("expected content to be stored, got %q", m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "Structured") {
		t.Fatalf("expected viewport to contain structured content, got %q", got)
	}
}

func TestPreviewNewModelWithContent(t *testing.T) {
	m := NewPreviewModel("# Structured", "Weekly Sync", 80, 20)

	if m.content != "# Structured" {
		t.Fatalf("expected content to be stored, got %q", m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "Structured") {
		t.Fatalf("expected viewport to contain structured content, got %q", got)
	}
}

func TestNewPreviewModelEmpty(t *testing.T) {
	m := NewPreviewModel("", "Weekly Sync", 80, 20)

	if m.content != previewPlaceholder {
		t.Fatalf("expected placeholder content %q, got %q", previewPlaceholder, m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "No structured notes yet") {
		t.Fatalf("expected viewport to contain placeholder, got %q", got)
	}
}

func TestPreviewNewModelEmptyUsesPlaceholder(t *testing.T) {
	m := NewPreviewModel("", "Weekly Sync", 80, 20)

	if m.content != previewPlaceholder {
		t.Fatalf("expected placeholder content %q, got %q", previewPlaceholder, m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "No structured notes yet") {
		t.Fatalf("expected viewport to contain placeholder, got %q", got)
	}
}

func TestPreviewInitReturnsNilCmd(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestPreviewEscReturnsToggle(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	updated, cmd := updatePreviewModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil command for esc")
	}

	msg := cmd()
	if _, ok := msg.(TogglePreviewMsg); !ok {
		t.Fatalf("expected TogglePreviewMsg from esc, got %T", msg)
	}

	if updated.title != "Title" {
		t.Fatalf("expected title to remain unchanged, got %q", updated.title)
	}
}

func TestPreviewCtrlPReturnsToggle(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	_, cmd := updatePreviewModelForTest(t, m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatalf("expected non-nil command for ctrl+p")
	}

	msg := cmd()
	if _, ok := msg.(TogglePreviewMsg); !ok {
		t.Fatalf("expected TogglePreviewMsg from ctrl+p, got %T", msg)
	}
}

func TestPreviewQReturnsToggle(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	_, cmd := updatePreviewModelForTest(t, m, tea.KeyPressMsg{Text: "q"})
	if cmd == nil {
		t.Fatalf("expected non-nil command for q")
	}

	msg := cmd()
	if _, ok := msg.(TogglePreviewMsg); !ok {
		t.Fatalf("expected TogglePreviewMsg from q, got %T", msg)
	}
}

func TestPreviewWindowResize(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	updated, cmd := updatePreviewModelForTest(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	if cmd != nil {
		t.Fatalf("expected nil command for window resize")
	}

	if updated.width != 120 || updated.height != 30 {
		t.Fatalf("expected model width=120 height=30, got width=%d height=%d", updated.width, updated.height)
	}

	if updated.viewport.Width() != 120 || updated.viewport.Height() != 28 {
		t.Fatalf("expected viewport width=120 height=28, got width=%d height=%d", updated.viewport.Width(), updated.viewport.Height())
	}
}

func TestPreviewViewContainsHeader(t *testing.T) {
	m := NewPreviewModel("content", "Sprint Review", 80, 20)

	view := m.View().Content
	if !strings.Contains(view, "Preview:") {
		t.Fatalf("expected view to contain Preview header, got %q", view)
	}

	if !strings.Contains(view, "Sprint Review") {
		t.Fatalf("expected view to contain title, got %q", view)
	}
}

func TestPreviewViewContainsFooter(t *testing.T) {
	m := NewPreviewModel("content", "Title", 80, 20)

	view := m.View().Content
	if !strings.Contains(view, "Esc") {
		t.Fatalf("expected view to contain Esc footer hint, got %q", view)
	}
}

func TestPreviewSetContent(t *testing.T) {
	m := NewPreviewModel("old", "Title", 80, 20)
	m.SetContent("new structured content")

	if m.content != "new structured content" {
		t.Fatalf("expected updated content, got %q", m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "new structured content") {
		t.Fatalf("expected viewport to contain updated content, got %q", got)
	}
}

func TestPreviewSetContentEmptyUsesPlaceholder(t *testing.T) {
	m := NewPreviewModel("old", "Title", 80, 20)
	m.SetContent("")

	if m.content != previewPlaceholder {
		t.Fatalf("expected placeholder content %q, got %q", previewPlaceholder, m.content)
	}

	if got := m.viewport.View(); !strings.Contains(got, "No structured notes yet") {
		t.Fatalf("expected viewport to contain placeholder after empty set, got %q", got)
	}
}
