package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func updateEmailModelForTest(t *testing.T, m EmailModel, msg tea.Msg) (EmailModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	emailModel, ok := updated.(EmailModel)
	if !ok {
		t.Fatalf("expected EmailModel from Update, got %T", updated)
	}

	return emailModel, cmd
}

func TestNewEmailModelWithContent(t *testing.T) {
	m := NewEmailModel("Weekly Sync Summary", "Hello team,\nMeeting notes attached.", 120, 30, "pl", "")

	if m.subject != "Weekly Sync Summary" {
		t.Fatalf("expected subject to be stored, got %q", m.subject)
	}

	if m.body != "Hello team,\nMeeting notes attached." {
		t.Fatalf("expected body to be stored, got %q", m.body)
	}

	view := m.viewport.View()
	if !strings.Contains(view, "Subject: Weekly Sync Summary") {
		t.Fatalf("expected viewport to contain formatted subject, got %q", view)
	}

	if !strings.Contains(view, "Meeting notes attached.") {
		t.Fatalf("expected viewport to contain email body, got %q", view)
	}
}

func TestNewEmailModelEmpty(t *testing.T) {
	m := NewEmailModel("", "", 120, 30, "pl", "")

	if m.subject != "" || m.body != "" {
		t.Fatalf("expected empty subject and body, got subject=%q body=%q", m.subject, m.body)
	}

	if got := m.viewport.View(); !strings.Contains(got, "No email generated yet") {
		t.Fatalf("expected viewport to contain placeholder, got %q", got)
	}
}

func TestEmailEscNavigates(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	_, cmd := updateEmailModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil command for esc")
	}

	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}

	if nav.Screen != ScreenMeetingList {
		t.Fatalf("expected ScreenMeetingList, got %v", nav.Screen)
	}
}

func TestEmailRReturnsRegenerate(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	_, cmd := updateEmailModelForTest(t, m, tea.KeyPressMsg{Text: "r"})
	if cmd == nil {
		t.Fatalf("expected non-nil command for r")
	}

	if _, ok := cmd().(RegenerateEmailMsg); !ok {
		t.Fatalf("expected RegenerateEmailMsg from r key")
	}
}

func TestEmailCCopiesClipboard(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	_, cmd := updateEmailModelForTest(t, m, tea.KeyPressMsg{Text: "c"})
	if cmd == nil {
		t.Fatalf("expected non-nil command for c")
	}
}

func TestEmailContentMsgUpdates(t *testing.T) {
	m := NewEmailModel("Old subject", "Old body", 120, 30, "pl", "")

	updated, cmd := updateEmailModelForTest(t, m, EmailContentMsg{Subject: "New subject", Body: "New body"})
	if cmd != nil {
		t.Fatalf("expected nil command for EmailContentMsg")
	}

	if updated.subject != "New subject" {
		t.Fatalf("expected updated subject, got %q", updated.subject)
	}

	if updated.body != "New body" {
		t.Fatalf("expected updated body, got %q", updated.body)
	}

	if got := updated.viewport.View(); !strings.Contains(got, "Subject: New subject") || !strings.Contains(got, "New body") {
		t.Fatalf("expected viewport to contain updated content, got %q", got)
	}
}

func TestEmailWindowResize(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 80, 20, "pl", "")

	updated, cmd := updateEmailModelForTest(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	if cmd != nil {
		t.Fatalf("expected nil command for window resize")
	}

	if updated.width != 140 || updated.height != 40 {
		t.Fatalf("expected model width=140 height=40, got width=%d height=%d", updated.width, updated.height)
	}

	if updated.viewport.Width() != 140 || updated.viewport.Height() != 38 {
		t.Fatalf("expected viewport width=140 height=38, got width=%d height=%d", updated.viewport.Width(), updated.viewport.Height())
	}
}

func TestEmailViewContainsHeader(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	view := m.View().Content
	if !strings.Contains(view, "Email Summary") {
		t.Fatalf("expected view to contain Email Summary header, got %q", view)
	}
}

func TestEmailViewContainsFooter(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	view := m.View().Content
	if !strings.Contains(view, "c copy") || !strings.Contains(view, "r regenerate") || !strings.Contains(view, "Esc back") {
		t.Fatalf("expected view to contain footer keybinding hints, got %q", view)
	}
}

func TestEmailClipboardDoneSetsStatus(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	updated, cmd := updateEmailModelForTest(t, m, ClipboardDoneMsg{})
	if cmd == nil {
		t.Fatalf("expected non-nil tick command for status clear")
	}

	if updated.statusMsg != "Copied to clipboard!" {
		t.Fatalf("expected copied status, got %q", updated.statusMsg)
	}
}

func TestEmailClipboardErrSetsStatus(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")

	updated, cmd := updateEmailModelForTest(t, m, ClipboardErrMsg{Err: errors.New("not available")})
	if cmd == nil {
		t.Fatalf("expected non-nil tick command for status clear")
	}

	if !strings.Contains(updated.statusMsg, "Clipboard unavailable:") {
		t.Fatalf("expected clipboard unavailable status prefix, got %q", updated.statusMsg)
	}

	if !strings.Contains(updated.statusMsg, "not available") {
		t.Fatalf("expected clipboard error detail in status, got %q", updated.statusMsg)
	}
}

func TestEmailClearStatusMsg(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 120, 30, "pl", "")
	m.statusMsg = "Copied to clipboard!"

	updated, cmd := updateEmailModelForTest(t, m, ClearStatusMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command for ClearStatusMsg")
	}

	if updated.statusMsg != "" {
		t.Fatalf("expected empty status after clear, got %q", updated.statusMsg)
	}
}

func TestEmailCopyEmptyContent(t *testing.T) {
	m := NewEmailModel("", "", 120, 30, "pl", "")

	updated, cmd := updateEmailModelForTest(t, m, tea.KeyPressMsg{Text: "c"})
	if cmd == nil {
		t.Fatalf("expected non-nil tick command for status clear")
	}

	if updated.statusMsg != "Nothing to copy" {
		t.Fatalf("expected 'Nothing to copy' status, got %q", updated.statusMsg)
	}
}
