package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
)

type staticOverlayContentModel struct {
	content string
}

func (m staticOverlayContentModel) Init() tea.Cmd { return nil }

func (m staticOverlayContentModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m staticOverlayContentModel) View() tea.View {
	return tea.NewView(m.content)
}

func openProviderOverlayFromEditor(t *testing.T) AppModel {
	t.Helper()

	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false, "")
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "", "")
	m.hasEditorModel = true

	opened, _ := appUpdate(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	return opened
}

func TestProviderOverlayCtrlOOpensOverlayWithoutScreenSwitch(t *testing.T) {
	opened := openProviderOverlayFromEditor(t)

	if opened.screen != ScreenEditor {
		t.Fatalf("expected screen to remain %v, got %v", ScreenEditor, opened.screen)
	}
	if opened.showProvider != true {
		t.Fatalf("expected showProvider true, got %v", opened.showProvider)
	}
	if opened.hasProviderModel != true {
		t.Fatalf("expected hasProviderModel true, got %v", opened.hasProviderModel)
	}
	if opened.providerModal.Title != "AI Provider" {
		t.Fatalf("expected provider modal title %q, got %q", "AI Provider", opened.providerModal.Title)
	}

	// While provider overlay is open, keys should route to provider modal,
	// not background app shortcuts such as help toggle.
	routed, _ := appUpdate(t, opened, tea.KeyPressMsg{Text: "?"})
	if routed.showProvider != true {
		t.Fatalf("expected showProvider to remain true after routed key, got %v", routed.showProvider)
	}
	if routed.showHelp != false {
		t.Fatalf("expected showHelp false when key is routed to provider modal, got %v", routed.showHelp)
	}
}

func TestProviderOverlayEscUsesTwoStepDismiss(t *testing.T) {
	opened := openProviderOverlayFromEditor(t)

	afterEsc, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from esc while provider overlay is showing")
	}
	if afterEsc.showProvider != true {
		t.Fatalf("expected showProvider true before DismissModalMsg, got %v", afterEsc.showProvider)
	}
	if afterEsc.screen != ScreenEditor {
		t.Fatalf("expected screen to stay %v before dismiss msg, got %v", ScreenEditor, afterEsc.screen)
	}

	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg, got %T", emitted)
	}

	dismissed, _ := appUpdate(t, afterEsc, emitted)
	if dismissed.showProvider != false {
		t.Fatalf("expected showProvider false after DismissModalMsg, got %v", dismissed.showProvider)
	}
	if dismissed.screen != ScreenEditor {
		t.Fatalf("expected screen to remain %v after dismiss, got %v", ScreenEditor, dismissed.screen)
	}
}

func TestProviderOverlayProviderSelectedClosesOverlay(t *testing.T) {
	opened := openProviderOverlayFromEditor(t)
	opened.providerFactory = func(*config.Config) (ai.Provider, error) {
		return appTestProvider{}, nil
	}

	updated, _ := appUpdate(t, opened, ProviderSelectedMsg{ProviderName: "openrouter"})

	if updated.showProvider != false {
		t.Fatalf("expected showProvider false after selection, got %v", updated.showProvider)
	}
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v after selection, got %v", ScreenEditor, updated.screen)
	}
	if updated.cfg.AIProvider != "openrouter" {
		t.Fatalf("expected cfg.AIProvider %q, got %q", "openrouter", updated.cfg.AIProvider)
	}
}

func TestProviderOverlayDismissModalMsgZOrderProviderThenNewMeetingThenHelp(t *testing.T) {
	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	m := NewAppModel(cfg, nil, appTestProvider{}, "", false, "")
	m.screen = ScreenEditor
	m.showHelp = true
	m.showNewMeeting = true
	m.showProvider = true
	m.helpModal = NewModalModel("Help", staticOverlayContentModel{content: "HELP_LAYER"}, 80, 24)
	m.newMeetingModal = NewModalModel("New Meeting", staticOverlayContentModel{content: "NEW_LAYER"}, 80, 24)
	m.providerModal = NewModalModel("AI Provider", staticOverlayContentModel{content: "PROVIDER_LAYER"}, 80, 24)

	afterFirstDismiss, _ := appUpdate(t, m, DismissModalMsg{})
	if afterFirstDismiss.showProvider != false {
		t.Fatalf("expected showProvider false after first dismiss, got %v", afterFirstDismiss.showProvider)
	}
	if afterFirstDismiss.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true after first dismiss, got %v", afterFirstDismiss.showNewMeeting)
	}
	if afterFirstDismiss.showHelp != true {
		t.Fatalf("expected showHelp true after first dismiss, got %v", afterFirstDismiss.showHelp)
	}

	afterSecondDismiss, _ := appUpdate(t, afterFirstDismiss, DismissModalMsg{})
	if afterSecondDismiss.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting false after second dismiss, got %v", afterSecondDismiss.showNewMeeting)
	}
	if afterSecondDismiss.showHelp != true {
		t.Fatalf("expected showHelp true after second dismiss, got %v", afterSecondDismiss.showHelp)
	}

	afterThirdDismiss, _ := appUpdate(t, afterSecondDismiss, DismissModalMsg{})
	if afterThirdDismiss.showHelp != false {
		t.Fatalf("expected showHelp false after third dismiss, got %v", afterThirdDismiss.showHelp)
	}
}

func TestProviderOverlayViewRendersProviderOnTopOfHelpAndNewMeeting(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "", false, "")
	m.screen = ScreenMeetingList
	m.width = 80
	m.height = 24
	m.showProvider = true
	m.providerModel = NewProviderModel("github_models", m.width, m.height)
	m.hasProviderModel = true
	m.providerModal = NewModalModel("AI Provider", &m.providerModel, m.width, m.height)

	view := m.View().Content
	if len(view) == 0 {
		t.Fatalf("expected non-empty view content when provider overlay is showing")
	}
	stripped := ansi.Strip(view)
	if !strings.Contains(stripped, "Select AI Provider") {
		t.Fatalf("expected view to contain provider list title 'Select AI Provider'")
	}
}

func TestProviderOverlayCtrlCQuitsWhenOverlayShowing(t *testing.T) {
	opened := openProviderOverlayFromEditor(t)

	_, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}
