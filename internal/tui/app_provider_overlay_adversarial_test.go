package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/config"
)

func newProviderOverlayAdversarialApp() AppModel {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false, "")
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "", "")
	m.hasEditorModel = true
	return m
}

func TestProviderOverlayAdversarial(t *testing.T) {
	t.Run("double ctrl+o open does not corrupt provider overlay state", func(t *testing.T) {
		m := newProviderOverlayAdversarialApp()

		firstOpen, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
		if firstOpen.showProvider != true {
			t.Fatalf("expected showProvider true after first ctrl+o, got %v", firstOpen.showProvider)
		}

		secondOpen, _ := appUpdate(t, firstOpen, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
		if secondOpen.showProvider != true {
			t.Fatalf("expected showProvider true after second ctrl+o, got %v", secondOpen.showProvider)
		}
		if secondOpen.screen != ScreenEditor {
			t.Fatalf("expected screen %v after second ctrl+o, got %v", ScreenEditor, secondOpen.screen)
		}
		if secondOpen.hasProviderModel != true {
			t.Fatalf("expected hasProviderModel true after second ctrl+o, got %v", secondOpen.hasProviderModel)
		}
		if secondOpen.providerModal.Title != "AI Provider" {
			t.Fatalf("expected provider modal title %q, got %q", "AI Provider", secondOpen.providerModal.Title)
		}
		if secondOpen.cfg.AIProvider != "github_models" {
			t.Fatalf("expected cfg.AIProvider unchanged %q, got %q", "github_models", secondOpen.cfg.AIProvider)
		}
	})

	t.Run("dismiss modal closes help when provider is not showing", func(t *testing.T) {
		m := NewAppModel(&config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}, nil, appTestProvider{}, "", false, "")
		m.screen = ScreenMeetingList
		m.showHelp = true
		m.showProvider = false

		updated, _ := appUpdate(t, m, DismissModalMsg{})

		if updated.showHelp != false {
			t.Fatalf("expected showHelp false after DismissModalMsg, got %v", updated.showHelp)
		}
		if updated.showProvider != false {
			t.Fatalf("expected showProvider false to remain unchanged, got %v", updated.showProvider)
		}
		if updated.screen != ScreenMeetingList {
			t.Fatalf("expected screen %v to remain unchanged, got %v", ScreenMeetingList, updated.screen)
		}
	})

	t.Run("zero-sized window while provider overlay open does not panic or corrupt state", func(t *testing.T) {
		m := newProviderOverlayAdversarialApp()
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
		if opened.showProvider != true {
			t.Fatalf("expected showProvider true before zero-sized resize, got %v", opened.showProvider)
		}

		updated, _ := appUpdate(t, opened, tea.WindowSizeMsg{Width: 0, Height: 0})

		if updated.width != 0 {
			t.Fatalf("expected width 0 after resize, got %d", updated.width)
		}
		if updated.height != 0 {
			t.Fatalf("expected height 0 after resize, got %d", updated.height)
		}
		if updated.showProvider != true {
			t.Fatalf("expected showProvider true after zero-sized resize, got %v", updated.showProvider)
		}
		if updated.screen != ScreenEditor {
			t.Fatalf("expected screen %v after zero-sized resize, got %v", ScreenEditor, updated.screen)
		}
	})

	t.Run("ctrl+o from non-editor screen opens provider overlay", func(t *testing.T) {
		m := NewAppModel(&config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}, nil, appTestProvider{}, "", false, "")
		m.screen = ScreenMeetingList

		updated, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})

		if updated.showProvider != true {
			t.Fatalf("expected showProvider true from non-editor ctrl+o, got %v", updated.showProvider)
		}
		if updated.hasProviderModel != true {
			t.Fatalf("expected hasProviderModel true from non-editor ctrl+o, got %v", updated.hasProviderModel)
		}
		if updated.screen != ScreenMeetingList {
			t.Fatalf("expected screen %v to remain unchanged, got %v", ScreenMeetingList, updated.screen)
		}
	})
}
