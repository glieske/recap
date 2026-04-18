package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func toConfirmModel(t *testing.T, updated tea.Model) ConfirmModel {
	t.Helper()

	cm, ok := updated.(ConfirmModel)
	if !ok {
		t.Fatalf("expected ConfirmModel, got %T", updated)
	}

	return cm
}

func toConfirmResultMsg(t *testing.T, cmd tea.Cmd) ConfirmResultMsg {
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

func TestConfirmAdversarial(t *testing.T) {
	t.Run("rapid toggle then enter keeps consistent focus", func(t *testing.T) {
		m := NewConfirmModel("Dangerous action?", "danger")

		sequence := []tea.KeyPressMsg{
			{Code: tea.KeyLeft},
			{Code: tea.KeyRight},
			{Code: tea.KeyLeft},
			{Code: tea.KeyRight},
		}

		for i, key := range sequence {
			updated, cmd := m.Update(key)
			m = toConfirmModel(t, updated)
			if cmd != nil {
				t.Fatalf("expected nil cmd at step %d, got non-nil", i)
			}
		}

		if m.focused != 0 {
			t.Fatalf("expected focused=0 after left-right-left-right, got %d", m.focused)
		}

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = toConfirmModel(t, updated)
		result := toConfirmResultMsg(t, cmd)

		if m.confirmed != true {
			t.Fatalf("expected confirmed=true, got %v", m.confirmed)
		}

		expected := ConfirmResultMsg{Confirmed: true, Action: "danger"}
		if result != expected {
			t.Fatalf("expected %#v, got %#v", expected, result)
		}
	})

	t.Run("empty question and action do not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic for empty question/action: %v", r)
			}
		}()

		m := NewConfirmModel("", "")
		if m.question != "" {
			t.Fatalf("expected empty question, got %q", m.question)
		}
		if m.action != "" {
			t.Fatalf("expected empty action, got %q", m.action)
		}

		if m.Init() != nil {
			t.Fatal("expected nil init command")
		}

		view := ansi.Strip(m.View().Content)
		if view != "\n\n[ Yes ] [ No ]" {
			t.Fatalf("expected exact stripped view for empty prompt, got %q", view)
		}
	})

	t.Run("very long question renders without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic for long question: %v", r)
			}
		}()

		longQ := strings.Repeat("A", 20000)
		m := NewConfirmModel(longQ, "long")

		if len(m.question) != 20000 {
			t.Fatalf("expected question length=20000, got %d", len(m.question))
		}

		stripped := ansi.Strip(m.View().Content)
		expectedPrefix := longQ + "\n\n"
		if stripped[:len(expectedPrefix)] != expectedPrefix {
			t.Fatal("expected stripped view to start with long question and spacing")
		}
	})

	t.Run("non keypress messages are no-op", func(t *testing.T) {
		m := NewConfirmModel("Proceed?", "noop")
		m.focused = 1
		m.confirmed = false

		updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		got := toConfirmModel(t, updated)

		if got != m {
			t.Fatalf("expected model unchanged for WindowSizeMsg; want %#v got %#v", m, got)
		}
		if cmd != nil {
			t.Fatalf("expected nil cmd for WindowSizeMsg, got %v", cmd)
		}
	})

	t.Run("double enter emits result message both times", func(t *testing.T) {
		m := NewConfirmModel("Proceed?", "double-enter")

		updated1, cmd1 := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m1 := toConfirmModel(t, updated1)
		result1 := toConfirmResultMsg(t, cmd1)

		expected1 := ConfirmResultMsg{Confirmed: true, Action: "double-enter"}
		if result1 != expected1 {
			t.Fatalf("first enter: expected %#v, got %#v", expected1, result1)
		}

		updated2, cmd2 := m1.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m2 := toConfirmModel(t, updated2)
		result2 := toConfirmResultMsg(t, cmd2)

		expected2 := ConfirmResultMsg{Confirmed: true, Action: "double-enter"}
		if result2 != expected2 {
			t.Fatalf("second enter: expected %#v, got %#v", expected2, result2)
		}

		if m2.confirmed != true {
			t.Fatalf("expected confirmed=true after second enter, got %v", m2.confirmed)
		}
	})
}
