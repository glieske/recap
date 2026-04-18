package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

func TestLegendAdversarial(t *testing.T) {
	t.Run("zero width does not panic", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "")
		m.width = 0

		legend := mustRenderLegendNoPanic(t, m)
		if strings.TrimSpace(legend) == "" {
			t.Fatalf("expected non-empty legend output for width=0")
		}
		if !strings.Contains(legend, "…") {
			t.Fatalf("expected width=0 output to include ellipsis, got %q", legend)
		}
	})

	t.Run("negative width does not panic", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "")
		m.width = -5

		legend := mustRenderLegendNoPanic(t, m)
		if strings.TrimSpace(legend) == "" {
			t.Fatalf("expected non-empty legend output for width=-5")
		}
		if !strings.Contains(legend, "…") {
			t.Fatalf("expected width=-5 output to include ellipsis, got %q", legend)
		}
	})

	t.Run("width one renders exactly ellipsis", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 1, 24, "", "")

		legend := mustRenderLegendNoPanic(t, m)
		if ansi.Strip(legend) != "…" {
			t.Fatalf("expected exactly %q for width=1, got %q", "…", legend)
		}
	})

	t.Run("width two truncates to one rune plus ellipsis", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 2, 24, "", "")

		legend := mustRenderLegendNoPanic(t, m)
		if ansi.Strip(legend) != "c…" {
			t.Fatalf("expected exactly %q for width=2, got %q", "c…", legend)
		}
		if utf8.RuneCountInString(ansi.Strip(legend)) != 2 {
			t.Fatalf("expected rune length 2, got %d in %q", utf8.RuneCountInString(legend), legend)
		}
	})

	t.Run("focused pane out of range in split mode does not panic", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "")
		m.splitMode = true
		m.focusedPane = 99

		legend := mustRenderLegendNoPanic(t, m)
		if strings.TrimSpace(legend) == "" {
			t.Fatalf("expected non-empty legend output for out-of-range focusedPane")
		}

		matchedKnownBranch := strings.Contains(legend, "ctrl+s Save | ctrl+a AI") ||
			strings.Contains(legend, "Tab Switch | ctrl+s Save | ctrl+a AI") ||
			strings.Contains(legend, "Tab Switch | ctrl+s Save Summary")
		if !matchedKnownBranch {
			t.Fatalf("expected legend to match one known branch, got %q", legend)
		}
	})

	t.Run("rapid split and focus toggling always renders valid legend", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 120, 24, "", "")
		focusValues := []int{-1, 0, 1, 2, 99}
		widthValues := []int{120, 2, 1, 0, -5}

		for i := 0; i < 500; i++ {
			m.splitMode = i%2 == 0
			m.focusedPane = focusValues[i%len(focusValues)]
			m.width = widthValues[i%len(widthValues)]

			legend := mustRenderLegendNoPanic(t, m)
			if strings.TrimSpace(legend) == "" {
				t.Fatalf("iteration %d: expected non-empty legend", i)
			}
			if m.width <= 1 && !strings.Contains(legend, "…") {
				t.Fatalf("iteration %d: expected ellipsis at width=%d, got %q", i, m.width, legend)
			}
		}
	})
}

func mustRenderLegendNoPanic(t *testing.T, m EditorModel) (legend string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderLegend panicked: %v", r)
		}
	}()

	return m.renderLegend()
}
