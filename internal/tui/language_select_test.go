package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func updateLanguageSelectModelForTest(t *testing.T, m *LanguageSelectModel, msg tea.Msg) (*LanguageSelectModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(*LanguageSelectModel)
	if !ok {
		t.Fatalf("expected *LanguageSelectModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func updateEmailModelForLanguageTest(t *testing.T, m EmailModel, msg tea.Msg) (EmailModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(EmailModel)
	if !ok {
		t.Fatalf("expected EmailModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestLanguageSelectModel_Navigation(t *testing.T) {
	m := NewLanguageSelectModel("pl")

	if m.cursor != 1 {
		t.Fatalf("expected initial cursor at index 1 for pl, got %d", m.cursor)
	}

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd on up key, got non-nil")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", updated.cursor)
	}

	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil cmd on down key, got non-nil")
	}
	if updated.cursor != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", updated.cursor)
	}

	updated.cursor = 0
	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd on wrapped up key, got non-nil")
	}
	if updated.cursor != 2 {
		t.Fatalf("expected cursor to wrap from 0 to 2 on up, got %d", updated.cursor)
	}

	updated.cursor = 2
	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil cmd on wrapped down key, got non-nil")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to wrap from 2 to 0 on down, got %d", updated.cursor)
	}
}

func TestLanguageSelectModel_SelectDifferentLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl")
	m.cursor = 0 // en

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd on enter")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to stay on selected option 0, got %d", updated.cursor)
	}

	emitted := cmd()
	langChanged, ok := emitted.(LanguageChangedMsg)
	if !ok {
		t.Fatalf("expected LanguageChangedMsg, got %T", emitted)
	}

	if langChanged.Language != "en" {
		t.Fatalf("expected selected language en, got %q", langChanged.Language)
	}
}

func TestLanguageSelectModel_SelectSameLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl")
	m.cursor = 1 // pl

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd on enter")
	}
	if updated.cursor != 1 {
		t.Fatalf("expected cursor to stay on index 1, got %d", updated.cursor)
	}

	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg when selecting current language, got %T", emitted)
	}
}

func TestLanguageSelectModel_NewModel_CursorPosition(t *testing.T) {
	m := NewLanguageSelectModel("pl")

	if m.cursor != 1 {
		t.Fatalf("expected cursor index 1 for pl, got %d", m.cursor)
	}
}

func TestLanguageSelectModel_InitReturnsNilCommand(t *testing.T) {
	m := NewLanguageSelectModel("en")

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestLanguageSelectModel_ViewMarksCursorAndCurrentLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl")

	view := m.View().Content
	stripped := ansi.Strip(view)
	if view == "" {
		t.Fatalf("expected non-empty view")
	}
	if !containsLine(stripped, "> PL *") {
		t.Fatalf("expected current selection marker in view, got %q", stripped)
	}
	if !containsLine(stripped, "  EN") {
		t.Fatalf("expected EN option in view, got %q", stripped)
	}
	if !containsLine(stripped, "  NO") {
		t.Fatalf("expected NO option in view, got %q", stripped)
	}
}

func containsLine(content, line string) bool {
	return content == line ||
		strings.Contains(content, "\n"+line+"\n") ||
		strings.HasPrefix(content, line+"\n") ||
		strings.HasSuffix(content, "\n"+line)
}

func TestEmailModel_LKeyEmitsShowLanguageModal(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 80, 24, "no", "")

	updated, cmd := updateEmailModelForLanguageTest(t, m, tea.KeyPressMsg{Text: "l"})
	if cmd == nil {
		t.Fatalf("expected non-nil command for l key")
	}
	if updated.language != "no" {
		t.Fatalf("expected language to remain unchanged in model, got %q", updated.language)
	}

	emitted := cmd()
	modalMsg, ok := emitted.(ShowLanguageModalMsg)
	if !ok {
		t.Fatalf("expected ShowLanguageModalMsg, got %T", emitted)
	}

	if modalMsg.CurrentLanguage != "no" {
		t.Fatalf("expected CurrentLanguage to be no, got %q", modalMsg.CurrentLanguage)
	}

	if _, isLanguageChanged := emitted.(LanguageChangedMsg); isLanguageChanged {
		t.Fatalf("did not expect LanguageChangedMsg from l key")
	}
}
