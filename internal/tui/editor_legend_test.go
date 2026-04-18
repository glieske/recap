package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestLegendNormalModeShowsDefaultBindings(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")

	legend := m.renderLegend()

	if !strings.Contains(legend, "ctrl+s Save | ctrl+a AI | ctrl+p Preview | ctrl+t Timestamp | ctrl+e Email | ctrl+o Provider | ctrl+/ Help") {
		t.Fatalf("expected normal legend bindings, got %q", legend)
	}
}

func TestLegendSplitModeLeftPaneShowsEditorBindings(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")
	m.splitMode = true
	m.focusedPane = 0

	legend := m.renderLegend()

	if !strings.Contains(legend, "Tab Switch | ctrl+s Save | ctrl+a AI | ctrl+t Timestamp | Esc Collapse | ctrl+/ Help") {
		t.Fatalf("expected split-left legend bindings, got %q", legend)
	}
}

func TestLegendSplitModeRightPaneShowsSummaryBindings(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 24, "", "")
	m.splitMode = true
	m.focusedPane = 1

	legend := m.renderLegend()

	if !strings.Contains(legend, "Tab Switch | ctrl+s Save Summary | ctrl+e Email | Esc Collapse | ctrl+/ Help") {
		t.Fatalf("expected split-right legend bindings, got %q", legend)
	}
}

func TestLegendTruncatesWithEllipsisWhenWidthIsNarrow(t *testing.T) {
	m := NewEditorModel(nil, nil, 12, 24, "", "")

	legend := m.renderLegend()

	if !strings.HasSuffix(ansi.Strip(legend), "…") {
		t.Fatalf("expected truncated legend to end with ellipsis, got %q", legend)
	}
	if strings.Contains(ansi.Strip(legend), "ctrl+/ Help") {
		t.Fatalf("expected narrow legend to truncate full text, got %q", legend)
	}
}

func TestLegendWidthOneRendersOnlyEllipsis(t *testing.T) {
	m := NewEditorModel(nil, nil, 1, 24, "", "")

	legend := m.renderLegend()

	if !strings.Contains(legend, "…") {
		t.Fatalf("expected width=1 legend to contain only ellipsis, got %q", legend)
	}
	if strings.Contains(legend, "ctrl+s") {
		t.Fatalf("expected width=1 legend to exclude keybinding text, got %q", legend)
	}
}
