package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
)

type settingsTestProvider struct {
	id string
}

func (p *settingsTestProvider) StructureNotes(context.Context, string, ai.MeetingMeta) (string, error) {
	return "", nil
}

func (p *settingsTestProvider) GenerateEmailSummary(context.Context, string, string) (string, error) {
	return "", nil
}

type settingsForwardedMsg struct{}

type settingsSpyModel struct {
	updates int
	lastMsg tea.Msg
	view    string
	emit    tea.Msg
}

func (m *settingsSpyModel) Init() tea.Cmd {
	return nil
}

func (m *settingsSpyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.updates++
	m.lastMsg = msg
	if m.emit == nil {
		return m, nil
	}

	return m, func() tea.Msg { return m.emit }
}

func (m *settingsSpyModel) View() tea.View {
	return tea.NewView(m.view)
}

func baseSettingsConfig() *config.Config {
	return &config.Config{
		NotesDir:         "~/recap",
		AIProvider:       "github_models",
		GitHubModel:      "gpt-4o",
		OpenRouterModel:  "google/gemini-flash-1.5",
		OpenRouterAPIKey: "",
		LMStudioURL:      "http://localhost:1234/v1",
		LMStudioModel:    "",
		EmailLanguage:    "en",
	}
}

func TestAppSettingsCtrlCommaOpensSettingsModal(t *testing.T) {
	testCases := []struct {
		name   string
		screen Screen
	}{
		{name: "meeting list", screen: ScreenMeetingList},
		{name: "editor", screen: ScreenEditor},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAppModel(baseSettingsConfig(), nil, nil, "", false, "")
			m.screen = tc.screen

			updated, cmd := appUpdate(t, m, tea.KeyPressMsg{Code: ',', Mod: tea.ModCtrl})

			if !updated.showSettings {
				t.Fatalf("expected showSettings true, got false")
			}
			if !updated.hasSettingsModel {
				t.Fatalf("expected hasSettingsModel true, got false")
			}
			if updated.settingsModal.Title != "Settings" {
				t.Fatalf("expected settings modal title %q, got %q", "Settings", updated.settingsModal.Title)
			}
			if cmd == nil {
				t.Fatalf("expected non-nil settings init command")
			}
		})
	}
}

func TestAppSettingsModalTrapsKeys(t *testing.T) {
	m := NewAppModel(baseSettingsConfig(), nil, nil, "", false, "")
	spy := &settingsSpyModel{emit: settingsForwardedMsg{}}
	m.showSettings = true
	m.settingsModal = NewModalModel("Settings", spy, m.width, m.height)

	updated, cmd := appUpdate(t, m, tea.KeyPressMsg{Text: "j"})

	if cmd == nil {
		t.Fatalf("expected command from settings modal update, got nil")
	}
	msg := cmd()
	if _, ok := msg.(settingsForwardedMsg); !ok {
		t.Fatalf("expected settingsForwardedMsg from settings modal, got %T", msg)
	}

	updatedSpy, ok := updated.settingsModal.Content.(*settingsSpyModel)
	if !ok {
		t.Fatalf("expected settings modal content type %T, got %T", &settingsSpyModel{}, updated.settingsModal.Content)
	}
	if updatedSpy.updates != 1 {
		t.Fatalf("expected settings modal update count 1, got %d", updatedSpy.updates)
	}
	if kp, ok := updatedSpy.lastMsg.(tea.KeyPressMsg); !ok || kp.Text != "j" {
		t.Fatalf("expected forwarded key message with Text %q, got %#v", "j", updatedSpy.lastMsg)
	}
}

func TestAppSettingsDismissModalClosesSettings(t *testing.T) {
	m := NewAppModel(baseSettingsConfig(), nil, nil, "", false, "")
	m.showSettings = true

	updated, _ := appUpdate(t, m, DismissModalMsg{})

	if updated.showSettings {
		t.Fatalf("expected showSettings false after dismiss, got true")
	}
}

func TestAppSettingsDismissModalRespectsZOrder(t *testing.T) {
	m := NewAppModel(baseSettingsConfig(), nil, nil, "", false, "")
	m.showProvider = true
	m.showSettings = true

	updated, _ := appUpdate(t, m, DismissModalMsg{})

	if updated.showProvider {
		t.Fatalf("expected showProvider false after dismiss, got true")
	}
	if !updated.showSettings {
		t.Fatalf("expected showSettings to remain true when provider overlay closes first")
	}
}

func TestAppSettingsUpdatedReloadsProviderAndSetsStatus(t *testing.T) {
	oldProvider := &settingsTestProvider{id: "old"}
	newProvider := &settingsTestProvider{id: "new"}
	updatedCfg := baseSettingsConfig()
	updatedCfg.AIProvider = "lm_studio"

	m := NewAppModel(baseSettingsConfig(), nil, oldProvider, "", false, "")
	m.showSettings = true

	factoryCalls := 0
	m.providerFactory = func(cfg *config.Config) (ai.Provider, error) {
		factoryCalls++
		if cfg != updatedCfg {
			t.Fatalf("expected providerFactory cfg pointer %p, got %p", updatedCfg, cfg)
		}
		return newProvider, nil
	}

	updated, _ := appUpdate(t, m, SettingsUpdatedMsg{Config: updatedCfg})

	if updated.showSettings {
		t.Fatalf("expected showSettings false after settings update, got true")
	}
	if factoryCalls != 1 {
		t.Fatalf("expected providerFactory to be called 1 time, got %d", factoryCalls)
	}
	if provider, ok := updated.provider.(*settingsTestProvider); !ok || provider.id != "new" {
		t.Fatalf("expected provider id %q, got %#v", "new", updated.provider)
	}
	if updated.statusMsg != "Settings saved" {
		t.Fatalf("expected statusMsg %q, got %q", "Settings saved", updated.statusMsg)
	}
	if updated.err != nil {
		t.Fatalf("expected err nil after successful settings update, got %v", updated.err)
	}
}

func TestAppSettingsUpdatedFactoryErrorPreservesProvider(t *testing.T) {
	oldProvider := &settingsTestProvider{id: "old"}
	updatedCfg := baseSettingsConfig()
	updateErr := errors.New("provider init failed")

	m := NewAppModel(baseSettingsConfig(), nil, oldProvider, "", false, "")
	m.providerFactory = func(*config.Config) (ai.Provider, error) {
		return nil, updateErr
	}

	updated, _ := appUpdate(t, m, SettingsUpdatedMsg{Config: updatedCfg})

	if provider, ok := updated.provider.(*settingsTestProvider); !ok || provider.id != "old" {
		t.Fatalf("expected old provider to be preserved, got %#v", updated.provider)
	}
	if updated.err == nil || updated.err.Error() != updateErr.Error() {
		t.Fatalf("expected err %q, got %v", updateErr.Error(), updated.err)
	}
	if !strings.Contains(updated.statusMsg, "provider error: provider init failed") {
		t.Fatalf("expected statusMsg to include provider error text, got %q", updated.statusMsg)
	}
}

func TestAppSettingsViewRendersSettingsOverlay(t *testing.T) {
	m := NewAppModel(baseSettingsConfig(), nil, nil, "", false, "")
	m.width = 200
	m.height = 40
	spy := &settingsSpyModel{view: "SOK"}
	m.showSettings = true
	m.settingsModal = NewModalModel("Settings", spy, m.width, m.height)

	rendered := ansi.Strip(m.View().Content)

	if rendered == "" {
		t.Fatalf("expected non-empty rendered view")
	}
	if !strings.Contains(rendered, "Settings") {
		t.Fatalf("expected rendered view to contain 'Settings' title, got %q", rendered)
	}
	if !strings.Contains(rendered, "SOK") {
		t.Fatalf("expected rendered view to contain spy content 'SOK', got %q", rendered)
	}
}
