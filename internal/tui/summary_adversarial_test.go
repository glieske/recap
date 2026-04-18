package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

func updateSummaryAdversarial(t *testing.T, m SummaryModel, msg tea.Msg) (SummaryModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(SummaryModel)
	if !ok {
		t.Fatalf("expected SummaryModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func mustNotPanic(t *testing.T, name string, fn func()) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		fn()
	})
}

func TestSummaryAdversarial(t *testing.T) {
	mustNotPanic(t, "zero-dimension model does not panic", func() {
		m := NewSummaryModel("", "PRJ", "meeting-1", nil, 0, 0)
		if m.width != 0 || m.height != 0 {
			t.Fatalf("expected width=0 height=0, got width=%d height=%d", m.width, m.height)
		}
		if got := m.Value(); got != "" {
			t.Fatalf("expected empty content, got %q", got)
		}
	})

	mustNotPanic(t, "negative dimensions via WindowSizeMsg do not panic", func() {
		m := NewSummaryModel("", "PRJ", "meeting-1", nil, 10, 5)
		updated, cmd := updateSummaryAdversarial(t, m, tea.WindowSizeMsg{Width: -1, Height: -1})
		if cmd != nil {
			t.Fatalf("expected nil cmd for WindowSizeMsg, got non-nil")
		}
		if updated.width != -1 || updated.height != -1 {
			t.Fatalf("expected width=-1 height=-1, got width=%d height=%d", updated.width, updated.height)
		}
	})

	mustNotPanic(t, "SetContent with 100KB payload does not panic", func() {
		payload := strings.Repeat("A", 100*1024)
		m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)
		m.SetContent(payload)

		if gotLen := len(m.Value()); gotLen != len(payload) {
			t.Fatalf("expected payload length %d, got %d", len(payload), gotLen)
		}
		if m.IsDirty() {
			t.Fatalf("expected dirty=false after SetContent on clean model")
		}
	})

	mustNotPanic(t, "SetContent empty string when clean resets value gracefully", func() {
		m := NewSummaryModel("seed", "PRJ", "meeting-1", nil, 80, 20)
		if m.IsDirty() {
			t.Fatalf("expected dirty=false at initialization")
		}

		m.SetContent("")
		if got := m.Value(); got != "" {
			t.Fatalf("expected empty value after SetContent(\"\"), got %q", got)
		}
		if m.IsOverwritePending() {
			t.Fatalf("expected no overwrite pending after clean SetContent")
		}
		if m.IsDirty() {
			t.Fatalf("expected dirty=false after clean SetContent")
		}
	})

	mustNotPanic(t, "rapid overwrite cycle accept then reject keeps consistent state", func() {
		m := NewSummaryModel("base", "PRJ", "meeting-1", nil, 80, 20)
		m.Focus()

		m1, _ := updateSummaryAdversarial(t, m, tea.KeyPressMsg{Text: "x"})
		if !m1.IsDirty() {
			t.Fatalf("expected dirty=true after first edit")
		}

		m1.SetContent("ai-1")
		if !m1.IsOverwritePending() {
			t.Fatalf("expected overwrite pending after first SetContent while dirty")
		}

		accepted, _ := updateSummaryAdversarial(t, m1, tea.KeyPressMsg{Text: "y"})
		if got := accepted.Value(); got != "ai-1" {
			t.Fatalf("expected accepted overwrite value %q, got %q", "ai-1", got)
		}
		if accepted.IsDirty() {
			t.Fatalf("expected dirty=false after accepting overwrite")
		}

		m2, _ := updateSummaryAdversarial(t, accepted, tea.KeyPressMsg{Text: "z"})
		if !m2.IsDirty() {
			t.Fatalf("expected dirty=true after second edit")
		}

		m2.SetContent("ai-2")
		if !m2.IsOverwritePending() {
			t.Fatalf("expected overwrite pending on second SetContent while dirty")
		}

		rejected, _ := updateSummaryAdversarial(t, m2, tea.KeyPressMsg{Text: "n"})
		if rejected.IsOverwritePending() {
			t.Fatalf("expected overwrite pending=false after rejection")
		}
		if !rejected.IsDirty() {
			t.Fatalf("expected dirty=true to remain after rejection")
		}
		if got := rejected.Value(); got != "ai-1z" {
			t.Fatalf("expected user-edited content preserved after rejection, got %q", got)
		}
	})

	mustNotPanic(t, "key input while unfocused and unchanged content keeps dirty false", func() {
		m := NewSummaryModel("baseline", "PRJ", "meeting-1", nil, 80, 20)
		m.Blur()

		updated, _ := updateSummaryAdversarial(t, m, tea.KeyPressMsg{Text: "q"})
		if got := updated.Value(); got != "baseline" {
			t.Fatalf("expected content unchanged while blurred, got %q", got)
		}
		if updated.IsDirty() {
			t.Fatalf("expected dirty=false when content is unchanged")
		}
	})

	mustNotPanic(t, "ctrl+s returns specific errors for nil store, empty project, empty meeting ID", func() {
		checkSaveErr := func(t *testing.T, m SummaryModel, want string) {
			t.Helper()

			updated, cmd := updateSummaryAdversarial(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
			if cmd == nil {
				t.Fatalf("expected non-nil save command")
			}

			msg := cmd()
			errMsg, ok := msg.(SummarySaveErrMsg)
			if !ok {
				t.Fatalf("expected SummarySaveErrMsg from save command, got %T", msg)
			}
			if errMsg.Err == nil {
				t.Fatalf("expected non-nil error in SummarySaveErrMsg")
			}
			if got := errMsg.Err.Error(); got != want {
				t.Fatalf("expected error %q, got %q", want, got)
			}

			updated, _ = updateSummaryAdversarial(t, updated, msg)
			if !strings.Contains(updated.View().Content, want) {
				t.Fatalf("expected status view to include %q", want)
			}
		}

		checkSaveErr(t, NewSummaryModel("data", "PRJ", "meeting-1", nil, 80, 20), "summary store is not configured")
		checkSaveErr(t, NewSummaryModel("data", "", "meeting-1", &storage.Store{}, 80, 20), "summary project is empty")
		checkSaveErr(t, NewSummaryModel("data", "PRJ", "", &storage.Store{}, 80, 20), "summary meeting ID is empty")
	})

	mustNotPanic(t, "rapid save done and save error messages do not panic", func() {
		m := NewSummaryModel("base", "PRJ", "meeting-1", nil, 80, 20)

		m1, _ := updateSummaryAdversarial(t, m, SummarySaveDoneMsg{})
		m2, _ := updateSummaryAdversarial(t, m1, SummarySaveErrMsg{Err: errors.New("e")})
		m3, _ := updateSummaryAdversarial(t, m2, SummarySaveDoneMsg{})
		m4, _ := updateSummaryAdversarial(t, m3, SummarySaveErrMsg{Err: errors.New("f")})

		if got := m4.statusMsg; got != "Save error: f" {
			t.Fatalf("expected last status message to win, got %q", got)
		}
	})

	mustNotPanic(t, "nil and unknown message types do not panic", func() {
		m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

		updated, _ := updateSummaryAdversarial(t, m, nil)
		if got := updated.Value(); got != "" {
			t.Fatalf("expected value unchanged after nil message, got %q", got)
		}

		updated, _ = updateSummaryAdversarial(t, updated, struct{ Name string }{Name: "unknown"})
		if got := updated.Value(); got != "" {
			t.Fatalf("expected value unchanged after unknown message, got %q", got)
		}
	})
}
