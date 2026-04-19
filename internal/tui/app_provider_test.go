package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
)

func TestAppProviderCtrlOOpensSelectorFromEditor(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
	m.hasEditorModel = true

	updated, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})

	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen to remain %v, got %v", ScreenEditor, updated.screen)
	}
	if !updated.showProvider {
		t.Fatalf("expected showProvider true, got false")
	}
	if !updated.hasProviderModel {
		t.Fatalf("expected hasProviderModel true, got false")
	}
}

func TestAppProviderCtrlOFromNonEditorDoesNotSwitchScreen(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.screen = ScreenMeetingList

	updated, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})

	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, updated.screen)
	}
	if !updated.hasProviderModel {
		t.Fatalf("expected hasProviderModel true, got false")
	}
}

func TestAppProviderSelectedMsgSuccessfulSwitchReturnsToEditor(t *testing.T) {
	cfg := &config.Config{
		AIProvider:    "github_models",
		GitHubModel:   "gpt-4o",
		LMStudioModel: "local-model",
	}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.screen = ScreenProviderSelector
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "GitHub Models", "gpt-4o")
	m.hasEditorModel = true

	updated, _ := appUpdate(t, m, ProviderSelectedMsg{ProviderName: "lm_studio"})

	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
	if updated.cfg.AIProvider != "lm_studio" {
		t.Fatalf("expected cfg.AIProvider %q, got %q", "lm_studio", updated.cfg.AIProvider)
	}
	if updated.statusMsg != "Switched to lm_studio" {
		t.Fatalf("expected statusMsg %q, got %q", "Switched to lm_studio", updated.statusMsg)
	}
	if updated.err != nil {
		t.Fatalf("expected err nil, got %v", updated.err)
	}
	if _, ok := updated.provider.(*ai.LMStudioProvider); !ok {
		t.Fatalf("expected provider type *ai.LMStudioProvider, got %T", updated.provider)
	}
	if updated.editorModel.providerName != "LM Studio" {
		t.Fatalf("expected editor providerName %q, got %q", "LM Studio", updated.editorModel.providerName)
	}
	if updated.editorModel.providerModel != "local-model" {
		t.Fatalf("expected editor providerModel %q, got %q", "local-model", updated.editorModel.providerModel)
	}
}

func TestAppProviderSelectedMsgFailedSwitchRevertsConfigAndProvider(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	oldProvider := appTestProvider{emailResponse: "existing"}
	m := NewAppModel(cfg, nil, oldProvider, "", false)
	m.screen = ScreenProviderSelector
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "GitHub Models", "gpt-4o")
	m.hasEditorModel = true
	m.providerFactory = func(*config.Config) (ai.Provider, error) {
		return nil, errors.New("mock provider failure")
	}

	updated, _ := appUpdate(t, m, ProviderSelectedMsg{ProviderName: "openrouter"})

	if updated.cfg.AIProvider != "github_models" {
		t.Fatalf("expected cfg.AIProvider reverted to %q, got %q", "github_models", updated.cfg.AIProvider)
	}
	if updated.provider != oldProvider {
		t.Fatalf("expected provider unchanged on failure, got %T", updated.provider)
	}
	if updated.err == nil {
		t.Fatalf("expected err to be set on failed provider switch")
	}
	if !strings.Contains(updated.statusMsg, "Failed to switch provider") {
		t.Fatalf("expected statusMsg to contain %q, got %q", "Failed to switch provider", updated.statusMsg)
	}
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
}

func TestAppProviderEscFromProviderSelectorReturnsToEditor(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.screen = ScreenEditor
	m.providerModel = NewProviderModel(cfg.AIProvider, 80, 24)
	m.hasProviderModel = true
	m.showProvider = true
	m.providerModal = NewModalModel("AI Provider", &m.providerModel, 80, 24)

	afterEsc, cmd := appUpdate(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !afterEsc.showProvider {
		t.Fatalf("expected showProvider to remain true before DismissModalMsg, got false")
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from modal esc")
	}
	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg, got %T", emitted)
	}
	dismissed, _ := appUpdate(t, afterEsc, emitted)

	if dismissed.showProvider {
		t.Fatalf("expected showProvider false after DismissModalMsg, got true")
	}
	if dismissed.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, dismissed.screen)
	}
}

func TestAppProviderQuestionMarkPassesThroughWithoutOpeningHelp(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.screen = ScreenEditor
	m.providerModel = NewProviderModel(cfg.AIProvider, 80, 24)
	m.hasProviderModel = true
	m.showProvider = true
	m.providerModal = NewModalModel("AI Provider", &m.providerModel, 80, 24)

	updated, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

	if !updated.showProvider {
		t.Fatalf("expected showProvider to remain true, got false")
	}
	if updated.showHelp {
		t.Fatalf("expected showHelp to remain false when provider is showing, got true")
	}
}

func TestAppProviderWindowSizePropagatesToProviderModel(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false)
	m.providerModel = NewProviderModel(cfg.AIProvider, 40, 10)
	m.hasProviderModel = true
	m.showProvider = true
	m.providerModal = NewModalModel("AI Provider", &m.providerModel, 40, 10)

	updated, _ := appUpdate(t, m, tea.WindowSizeMsg{Width: 101, Height: 33})

	var innerWidth, innerHeight int
	switch p := updated.providerModal.Content.(type) {
	case *ProviderModel:
		innerWidth = p.width
		innerHeight = p.height
	case ProviderModel:
		innerWidth = p.width
		innerHeight = p.height
	default:
		t.Fatalf("expected providerModal.Content to be ProviderModel, got %T", updated.providerModal.Content)
	}
	if innerWidth != 101 {
		t.Fatalf("expected providerModel.width %d, got %d", 101, innerWidth)
	}
	if innerHeight != 33 {
		t.Fatalf("expected providerModel.height %d, got %d", 33, innerHeight)
	}
}
