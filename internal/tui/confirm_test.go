package tui

import (
	"regexp"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func updateConfirmModel(t *testing.T, m ConfirmModel, msg tea.Msg) (ConfirmModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(ConfirmModel)
	if !ok {
		t.Fatalf("expected ConfirmModel, got %T", updated)
	}

	return updatedModel, cmd
}

func cmdResultMsg(t *testing.T, cmd tea.Cmd) ConfirmResultMsg {
	t.Helper()

	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	if !ok {
		t.Fatalf("expected ConfirmResultMsg, got %T", msg)
	}

	return result
}

func TestConfirmNewConfirmModelDefaults(t *testing.T) {
	m := NewConfirmModel("Delete item?", "delete-item")

	if m.question != "Delete item?" {
		t.Fatalf("expected question %q, got %q", "Delete item?", m.question)
	}

	if m.action != "delete-item" {
		t.Fatalf("expected action %q, got %q", "delete-item", m.action)
	}

	if m.focused != 0 {
		t.Fatalf("expected focused=0, got %d", m.focused)
	}
}

func TestConfirmInitReturnsNil(t *testing.T) {
	m := NewConfirmModel("Proceed?", "proceed")

	cmd := m.Init()
	if cmd != nil {
		t.Fatalf("expected nil init cmd, got %v", cmd)
	}
}

func TestConfirmUpdateYConfirmsTrue(t *testing.T) {
	m := NewConfirmModel("Proceed?", "send-email")

	updated, cmd := updateConfirmModel(t, m, tea.KeyPressMsg{Text: "y"})
	result := cmdResultMsg(t, cmd)

	if !updated.confirmed {
		t.Fatal("expected updated.confirmed=true")
	}

	expected := ConfirmResultMsg{Confirmed: true, Action: "send-email"}
	if result != expected {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}
}

func TestConfirmUpdateNCancelsFalse(t *testing.T) {
	m := NewConfirmModel("Proceed?", "send-email")

	updated, cmd := updateConfirmModel(t, m, tea.KeyPressMsg{Text: "n"})
	result := cmdResultMsg(t, cmd)

	if updated.confirmed {
		t.Fatal("expected updated.confirmed=false")
	}

	expected := ConfirmResultMsg{Confirmed: false, Action: "send-email"}
	if result != expected {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}
}

func TestConfirmUpdateEnterFocusedYesReturnsConfirmed(t *testing.T) {
	m := NewConfirmModel("Proceed?", "archive")
	m.focused = 0

	updated, cmd := updateConfirmModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	result := cmdResultMsg(t, cmd)

	if !updated.confirmed {
		t.Fatal("expected updated.confirmed=true")
	}

	expected := ConfirmResultMsg{Confirmed: true, Action: "archive"}
	if result != expected {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}
}

func TestConfirmUpdateEnterFocusedNoReturnsCancelled(t *testing.T) {
	m := NewConfirmModel("Proceed?", "archive")
	m.focused = 1

	updated, cmd := updateConfirmModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	result := cmdResultMsg(t, cmd)

	if updated.confirmed {
		t.Fatal("expected updated.confirmed=false")
	}

	expected := ConfirmResultMsg{Confirmed: false, Action: "archive"}
	if result != expected {
		t.Fatalf("expected %#v, got %#v", expected, result)
	}
}

func TestConfirmUpdateLeftRightTabToggleFocus(t *testing.T) {
	m := NewConfirmModel("Proceed?", "archive")

	updatedLeft, cmdLeft := updateConfirmModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if updatedLeft.focused != 1 {
		t.Fatalf("expected focused=1 after left, got %d", updatedLeft.focused)
	}
	if cmdLeft != nil {
		t.Fatalf("expected nil cmd for left key, got %v", cmdLeft)
	}

	updatedRight, cmdRight := updateConfirmModel(t, updatedLeft, tea.KeyPressMsg{Code: tea.KeyRight})
	if updatedRight.focused != 0 {
		t.Fatalf("expected focused=0 after right, got %d", updatedRight.focused)
	}
	if cmdRight != nil {
		t.Fatalf("expected nil cmd for right key, got %v", cmdRight)
	}

	updatedTab, cmdTab := updateConfirmModel(t, updatedRight, tea.KeyPressMsg{Code: tea.KeyTab})
	if updatedTab.focused != 1 {
		t.Fatalf("expected focused=1 after tab, got %d", updatedTab.focused)
	}
	if cmdTab != nil {
		t.Fatalf("expected nil cmd for tab key, got %v", cmdTab)
	}
}

func TestConfirmViewRendersQuestionAndButtons(t *testing.T) {
	m := NewConfirmModel("Delete all notes?", "delete-all")

	view := m.View().Content
	stripped := ansi.Strip(view)

	if stripped != "Delete all notes?\n\n[ Yes ] [ No ]" {
		t.Fatalf("expected exact stripped view %q, got %q", "Delete all notes?\\n\\n[ Yes ] [ No ]", stripped)
	}
}

func TestConfirmViewFocusedButtonIsBold(t *testing.T) {
	boldYesPattern := regexp.MustCompile(`\x1b\[[0-9;]*m\[ Yes \]\x1b\[[0-9;]*m`)
	boldNoPattern := regexp.MustCompile(`\x1b\[[0-9;]*m\[ No \]\x1b\[[0-9;]*m`)

	yesFocused := NewConfirmModel("Proceed?", "act")
	if !boldYesPattern.MatchString(yesFocused.View().Content) {
		t.Fatalf("expected bold ANSI styling around [ Yes ], got %q", yesFocused.View().Content)
	}

	noFocused := NewConfirmModel("Proceed?", "act")
	noFocused.focused = 1
	if !boldNoPattern.MatchString(noFocused.View().Content) {
		t.Fatalf("expected bold ANSI styling around [ No ], got %q", noFocused.View().Content)
	}
}

func TestConfirmUnknownKeyIsNoOp(t *testing.T) {
	m := NewConfirmModel("Proceed?", "op")
	m.confirmed = false
	m.focused = 1

	updated, cmd := updateConfirmModel(t, m, tea.KeyPressMsg{Text: "z"})

	if updated != m {
		t.Fatalf("expected unchanged model %#v, got %#v", m, updated)
	}

	if cmd != nil {
		t.Fatalf("expected nil cmd for unknown key, got %v", cmd)
	}
}
