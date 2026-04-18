package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func providerItemsFromModel(t *testing.T, m ProviderModel) []ProviderItem {
	t.Helper()

	rawItems := m.list.Items()
	items := make([]ProviderItem, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(ProviderItem)
		if !ok {
			t.Fatalf("expected ProviderItem, got %T", raw)
		}
		items = append(items, item)
	}

	return items
}

func updateProviderModelForTest(t *testing.T, m ProviderModel, msg tea.Msg) (ProviderModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(ProviderModel)
	if !ok {
		t.Fatalf("expected ProviderModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestProviderNewProviderModelMarksCurrentProvider(t *testing.T) {
	cases := []struct {
		name            string
		currentProvider string
		expectedTitle   string
	}{
		{name: "github models", currentProvider: "github_models", expectedTitle: "✓ GitHub Models"},
		{name: "openrouter", currentProvider: "openrouter", expectedTitle: "✓ OpenRouter"},
		{name: "lm studio", currentProvider: "lm_studio", expectedTitle: "✓ LM Studio"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewProviderModel(tc.currentProvider, 80, 24)
			items := providerItemsFromModel(t, m)

			if len(items) != 3 {
				t.Fatalf("expected 3 provider items, got %d", len(items))
			}

			checkedCount := 0
			for _, item := range items {
				if strings.HasPrefix(item.Title(), "✓ ") {
					checkedCount++
				}
			}

			if checkedCount != 1 {
				t.Fatalf("expected exactly one current provider, got %d", checkedCount)
			}

			foundExpected := false
			for _, item := range items {
				if item.Title() == tc.expectedTitle {
					foundExpected = true
					break
				}
			}

			if !foundExpected {
				t.Fatalf("expected title %q to be marked current", tc.expectedTitle)
			}
		})
	}
}

func TestProviderItemTitleUsesCurrentPrefix(t *testing.T) {
	current := ProviderItem{displayName: "GitHub Models", current: true}
	notCurrent := ProviderItem{displayName: "OpenRouter", current: false}

	if got := current.Title(); got != "✓ GitHub Models" {
		t.Fatalf("expected checked title, got %q", got)
	}

	if got := notCurrent.Title(); got != "  OpenRouter" {
		t.Fatalf("expected unchecked title, got %q", got)
	}
}

func TestProviderItemDescriptionReturnsExpectedText(t *testing.T) {
	item := ProviderItem{description: "OpenRouter API (requires API key)"}

	if got := item.Description(); got != "OpenRouter API (requires API key)" {
		t.Fatalf("expected description %q, got %q", "OpenRouter API (requires API key)", got)
	}
}

func TestProviderItemFilterValueReturnsDisplayName(t *testing.T) {
	item := ProviderItem{displayName: "LM Studio"}

	if got := item.FilterValue(); got != "LM Studio" {
		t.Fatalf("expected filter value %q, got %q", "LM Studio", got)
	}
}

func TestProviderInitReturnsNil(t *testing.T) {
	m := NewProviderModel("github_models", 80, 24)

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestProviderUpdateEscReturnsNavigateToEditor(t *testing.T) {
	m := NewProviderModel("github_models", 80, 24)

	_, cmd := updateProviderModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil command for esc")
	}

	msg := cmd()
	navigate, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}

	if navigate.Screen != ScreenEditor {
		t.Fatalf("expected navigation to ScreenEditor, got %v", navigate.Screen)
	}
}

func TestProviderUpdateEnterReturnsSelectedProvider(t *testing.T) {
	m := NewProviderModel("openrouter", 80, 24)

	_, cmd := updateProviderModelForTest(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil command for enter")
	}

	msg := cmd()
	selected, ok := msg.(ProviderSelectedMsg)
	if !ok {
		t.Fatalf("expected ProviderSelectedMsg, got %T", msg)
	}

	if selected.ProviderName != "github_models" {
		t.Fatalf("expected selected provider %q, got %q", "github_models", selected.ProviderName)
	}
}

func TestProviderUpdateWindowSizeUpdatesDimensions(t *testing.T) {
	m := NewProviderModel("github_models", 40, 10)

	updated, cmd := updateProviderModelForTest(t, m, tea.WindowSizeMsg{Width: 120, Height: 42})
	if cmd != nil {
		t.Fatalf("expected nil command for window size update")
	}

	if updated.width != 120 || updated.height != 42 {
		t.Fatalf("expected width=120 height=42, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestProviderViewReturnsNonEmptyString(t *testing.T) {
	m := NewProviderModel("github_models", 80, 24)

	if got := m.View().Content; got == "" {
		t.Fatalf("expected non-empty view")
	}
}
