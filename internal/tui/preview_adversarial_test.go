package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func assertNoPanicPreview(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got panic: %v", r)
		}
	}()

	fn()
}

func TestPreviewAdversarial_ZeroDimensions_NoPanic(t *testing.T) {
	var m PreviewModel

	assertNoPanicPreview(t, func() {
		m = NewPreviewModel("content", "Zero Size", 0, 0)
	})

	if m.width != 0 || m.height != 0 {
		t.Fatalf("expected width=0 and height=0, got width=%d height=%d", m.width, m.height)
	}

	if m.viewport.Height() != 1 {
		t.Fatalf("expected viewport height to clamp to 1, got %d", m.viewport.Height())
	}

	if !strings.Contains(m.View().Content, "Zero Size") {
		t.Fatalf("expected rendered view to contain title %q", "Zero Size")
	}
}

func TestPreviewAdversarial_NegativeDimensions_NoPanic(t *testing.T) {
	m := NewPreviewModel("content", "Negative Size", -1, -1)

	if m.width != -1 || m.height != -1 {
		t.Fatalf("expected width=-1 and height=-1, got width=%d height=%d", m.width, m.height)
	}

	if m.viewport.Height() != 1 {
		t.Fatalf("expected viewport height to clamp to 1, got %d", m.viewport.Height())
	}

	assertNoPanicPreview(t, func() {
		updated, cmd := m.Update(tea.WindowSizeMsg{Width: -999, Height: -999})
		if cmd != nil {
			t.Fatalf("expected nil command for resize, got non-nil")
		}

		previewModel, ok := updated.(PreviewModel)
		if !ok {
			t.Fatalf("expected PreviewModel from Update, got %T", updated)
		}

		if previewModel.height != -999 {
			t.Fatalf("expected updated height=-999, got %d", previewModel.height)
		}
	})
}

func TestPreviewAdversarial_VeryLargeContent_NoPanic(t *testing.T) {
	large := strings.Repeat("A", 100*1024)

	var m PreviewModel
	assertNoPanicPreview(t, func() {
		m = NewPreviewModel(large, "Large Payload", 80, 20)
	})

	if got := len(m.content); got != len(large) {
		t.Fatalf("expected stored content length %d, got %d", len(large), got)
	}

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestPreviewAdversarial_UnicodeContent_NoPanic(t *testing.T) {
	unicodeContent := "emoji 😀 你好 مرحبا שָׁלוֹם\nRTL: \u202E\ncombining: e\u0301"

	var m PreviewModel
	assertNoPanicPreview(t, func() {
		m = NewPreviewModel(unicodeContent, "Unicode", 80, 20)
	})

	if m.content != unicodeContent {
		t.Fatalf("expected unicode content to be preserved exactly")
	}

	if !strings.Contains(m.View().Content, "Unicode") {
		t.Fatalf("expected rendered view to contain title %q", "Unicode")
	}
}

func TestPreviewAdversarial_SetContentVeryLongSingleLine_NoPanic(t *testing.T) {
	m := NewPreviewModel("seed", "Long Line", 80, 20)
	line := strings.Repeat("X", 120*1024)

	assertNoPanicPreview(t, func() {
		m.SetContent(line)
	})

	if got := len(m.content); got != len(line) {
		t.Fatalf("expected stored content length %d, got %d", len(line), got)
	}
}

func TestPreviewAdversarial_RapidSequentialSetContent_NoPanic(t *testing.T) {
	m := NewPreviewModel("seed", "Rapid", 80, 20)
	final := ""

	assertNoPanicPreview(t, func() {
		for i := range 2000 {
			payload := fmt.Sprintf("iteration=%d <script>alert(1)</script> ../ ..\\ ..%%2F", i)
			m.SetContent(payload)
			final = payload
		}
	})

	if m.content != final {
		t.Fatalf("expected final content to match last payload")
	}
}

func TestPreviewAdversarial_ANSISequences_NoPanic(t *testing.T) {
	ansiContent := "\x1b[31mred\x1b[0m\n\x1b[2Jclear\n\x1b[1;1Hhome"

	var m PreviewModel
	assertNoPanicPreview(t, func() {
		m = NewPreviewModel(ansiContent, "ANSI", 80, 20)
	})

	if m.content != ansiContent {
		t.Fatalf("expected ANSI content to be stored exactly")
	}

	assertNoPanicPreview(t, func() {
		m.SetContent(ansiContent)
	})

	if !strings.Contains(m.View().Content, "ANSI") {
		t.Fatalf("expected rendered view to contain title %q", "ANSI")
	}
}
