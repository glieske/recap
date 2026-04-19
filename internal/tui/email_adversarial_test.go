package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func assertNotPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got panic: %v", r)
		}
	}()

	fn()
}

func TestEmailAdversarial(t *testing.T) {
	t.Run("init on zero-dimension model returns nil command", func(t *testing.T) {
		m := NewEmailModel("subject", "body", 0, 0, "pl", "")

		assertNotPanics(t, func() {
			cmd := m.Init()
			if cmd != nil {
				t.Fatalf("expected nil init command, got non-nil")
			}
		})
	})

	t.Run("zero dimensions constructor does not panic", func(t *testing.T) {
		assertNotPanics(t, func() {
			m := NewEmailModel("subject", "body", 0, 0, "pl", "")
			if m.viewport.Width() != 0 {
				t.Fatalf("expected width 0, got %d", m.viewport.Width())
			}
			if m.viewport.Height() != 1 {
				t.Fatalf("expected clamped viewport height 1, got %d", m.viewport.Height())
			}
		})
	})

	t.Run("negative dimensions constructor does not panic", func(t *testing.T) {
		assertNotPanics(t, func() {
			m := NewEmailModel("subject", "body", -1, -1, "pl", "")
			if m.viewport.Width() != -1 {
				t.Fatalf("expected width -1, got %d", m.viewport.Width())
			}
			if m.viewport.Height() != 1 {
				t.Fatalf("expected clamped viewport height 1, got %d", m.viewport.Height())
			}
		})
	})

	t.Run("very large body 100KB does not panic", func(t *testing.T) {
		largeBody := strings.Repeat("A", 100*1024)

		assertNotPanics(t, func() {
			m := NewEmailModel("large", largeBody, 120, 20, "pl", "")
			if len(m.body) != 100*1024 {
				t.Fatalf("expected body length 102400, got %d", len(m.body))
			}
			view := m.View().Content
			if !strings.Contains(view, "Subject: large") {
				t.Fatalf("expected view to contain subject prefix")
			}
		})
	})

	t.Run("unicode subject and body render without panic", func(t *testing.T) {
		subject := "🚀 Sprint 更新 — שלום"
		body := "CJK: 進捗 ✅\nRTL: مرحبا بالعالم\nEmoji: 😀🔥"

		assertNotPanics(t, func() {
			m := NewEmailModel(subject, body, 120, 20, "pl", "")
			view := m.View().Content
			if !strings.Contains(view, "Subject: "+subject) {
				t.Fatalf("expected unicode subject to be present in view")
			}
			if !strings.Contains(view, "CJK: 進捗 ✅") {
				t.Fatalf("expected unicode body fragment to be present in view")
			}
		})
	})

	t.Run("subject with embedded newlines does not break formatting", func(t *testing.T) {
		subject := "line1\nline2"
		body := "body"

		assertNotPanics(t, func() {
			m := NewEmailModel(subject, body, 120, 20, "pl", "")
			view := m.View().Content
			if !strings.Contains(view, "Subject: line1") {
				t.Fatalf("expected first subject line in view")
			}
			if !strings.Contains(view, "line2") {
				t.Fatalf("expected second subject line in view")
			}
			if !strings.Contains(view, "body") {
				t.Fatalf("expected body content in view")
			}
		})
	})

	t.Run("EmailContentMsg with empty subject and non-empty body", func(t *testing.T) {
		m := NewEmailModel("initial", "initial", 120, 20, "pl", "")

		assertNotPanics(t, func() {
			updated, _ := m.Update(EmailContentMsg{Subject: "", Body: "body-only"})
			emailModel, ok := updated.(EmailModel)
			if !ok {
				t.Fatalf("expected EmailModel type after update")
			}
			if emailModel.subject != "" {
				t.Fatalf("expected empty subject, got %q", emailModel.subject)
			}
			if emailModel.body != "body-only" {
				t.Fatalf("expected body-only body, got %q", emailModel.body)
			}
			if !strings.Contains(emailModel.View().Content, "Subject: ") {
				t.Fatalf("expected subject label to be present")
			}
		})
	})

	t.Run("rapid ClearStatusMsg sequence does not panic", func(t *testing.T) {
		m := NewEmailModel("subject", "body", 120, 20, "pl", "")
		m.statusMsg = "temporary status"

		assertNotPanics(t, func() {
			for i := 0; i < 100; i++ {
				updated, _ := m.Update(ClearStatusMsg{})
				emailModel, ok := updated.(EmailModel)
				if !ok {
					t.Fatalf("expected EmailModel type after ClearStatusMsg update")
				}
				m = emailModel
			}
			if m.statusMsg != "" {
				t.Fatalf("expected cleared status message, got %q", m.statusMsg)
			}
		})
	})

	t.Run("window size update with negative dimensions remains stable", func(t *testing.T) {
		m := NewEmailModel("subject", "body", 80, 10, "pl", "")

		assertNotPanics(t, func() {
			updated, _ := m.Update(tea.WindowSizeMsg{Width: -50, Height: -9})
			emailModel, ok := updated.(EmailModel)
			if !ok {
				t.Fatalf("expected EmailModel type after window size update")
			}
			if emailModel.viewport.Height() != 1 {
				t.Fatalf("expected clamped viewport height 1, got %d", emailModel.viewport.Height())
			}
		})
	})
}
