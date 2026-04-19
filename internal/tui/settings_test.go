package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/config"
)

func TestSettingsModel_GroupCount(t *testing.T) {
	model := NewSettingsModel(&config.Config{AIProvider: "github_models"}, "", 80, 24)
	if model.form == nil {
		t.Fatalf("expected settings form to be initialized")
	}

	initCmd := model.Init()
	if initCmd == nil {
		t.Fatalf("expected init command to be non-nil")
	}
}

func TestSettingsModel_ValuesPreservedAcrossProviders(t *testing.T) {
	values := &settingsFormValues{
		provider:         "openrouter",
		githubModel:      "gpt-4o",
		openrouterModel:  "claude-3",
		openrouterAPIKey: "sk-test",
	}

	form := buildSettingsForm(values)
	if form == nil {
		t.Fatalf("expected form to be non-nil")
	}

	if values.githubModel != "gpt-4o" {
		t.Fatalf("expected githubModel to remain %q, got %q", "gpt-4o", values.githubModel)
	}
	if values.openrouterModel != "claude-3" {
		t.Fatalf("expected openrouterModel to remain %q, got %q", "claude-3", values.openrouterModel)
	}
	if values.openrouterAPIKey != "sk-test" {
		t.Fatalf("expected openrouterAPIKey to remain %q, got %q", "sk-test", values.openrouterAPIKey)
	}
}

func TestSettingsModel_DefaultsToGitHubModels(t *testing.T) {
	model := NewSettingsModel(nil, "", 80, 24)

	if model.values == nil {
		t.Fatalf("expected model values to be initialized")
	}
	if model.values.provider != "github_models" {
		t.Fatalf("expected provider %q, got %q", "github_models", model.values.provider)
	}
}

func TestSettingsModel_OpenRouterProvider(t *testing.T) {
	model := NewSettingsModel(&config.Config{
		AIProvider:      "openrouter",
		OpenRouterModel: "claude-3",
	}, "", 80, 24)

	if model.values == nil {
		t.Fatalf("expected model values to be initialized")
	}
	if model.values.provider != "openrouter" {
		t.Fatalf("expected provider %q, got %q", "openrouter", model.values.provider)
	}
	if model.values.openrouterModel != "claude-3" {
		t.Fatalf("expected openrouterModel %q, got %q", "claude-3", model.values.openrouterModel)
	}
}

func TestSettingsModel_LMStudioProvider(t *testing.T) {
	model := NewSettingsModel(&config.Config{
		AIProvider:    "lm_studio",
		LMStudioURL:   "http://localhost:1234",
		LMStudioModel: "local-model",
	}, "", 80, 24)

	if model.values == nil {
		t.Fatalf("expected model values to be initialized")
	}
	if model.values.provider != "lm_studio" {
		t.Fatalf("expected provider %q, got %q", "lm_studio", model.values.provider)
	}
	if model.values.lmStudioURL != "http://localhost:1234" {
		t.Fatalf("expected lmStudioURL %q, got %q", "http://localhost:1234", model.values.lmStudioURL)
	}
	if model.values.lmStudioModel != "local-model" {
		t.Fatalf("expected lmStudioModel %q, got %q", "local-model", model.values.lmStudioModel)
	}
}

func TestSettingsModel_ConfigFromValues_ValidAndTrimmed(t *testing.T) {
	model := NewSettingsModel(nil, "", 80, 24)
	model.values = &settingsFormValues{
		provider:         " openrouter ",
		githubModel:      " gpt-4o ",
		openrouterModel:  " claude-3 ",
		openrouterAPIKey: " sk-test ",
		lmStudioURL:      " https://localhost:1234 ",
		lmStudioModel:    " local ",
		emailLanguage:    " en ",
		notesDir:         " /tmp/notes ",
	}

	cfg, err := model.configFromValues()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.AIProvider != "openrouter" {
		t.Fatalf("expected AIProvider %q, got %q", "openrouter", cfg.AIProvider)
	}
	if cfg.OpenRouterModel != "claude-3" {
		t.Fatalf("expected OpenRouterModel %q, got %q", "claude-3", cfg.OpenRouterModel)
	}
	if cfg.OpenRouterAPIKey != "sk-test" {
		t.Fatalf("expected OpenRouterAPIKey %q, got %q", "sk-test", cfg.OpenRouterAPIKey)
	}
	if cfg.NotesDir != "/tmp/notes" {
		t.Fatalf("expected NotesDir %q, got %q", "/tmp/notes", cfg.NotesDir)
	}
}

func TestSettingsModel_ConfigFromValues_InvalidProvider(t *testing.T) {
	model := NewSettingsModel(nil, "", 80, 24)
	model.values = &settingsFormValues{provider: "invalid", notesDir: "/tmp/notes"}

	_, err := model.configFromValues()
	if err == nil {
		t.Fatalf("expected error for invalid provider")
	}
	if err.Error() != "invalid AI provider: invalid" {
		t.Fatalf("expected error %q, got %q", "invalid AI provider: invalid", err.Error())
	}
}

func TestSettingsModel_UpdateWindowSizeMutatesModel(t *testing.T) {
	model := NewSettingsModel(nil, "", 80, 24)

	updatedAny, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	updated, ok := updatedAny.(SettingsModel)
	if !ok {
		t.Fatalf("expected updated model type %T, got %T", SettingsModel{}, updatedAny)
	}
	if updated.width != 120 {
		t.Fatalf("expected width %d, got %d", 120, updated.width)
	}
	if updated.height != 35 {
		t.Fatalf("expected height %d, got %d", 35, updated.height)
	}
}

func TestSettingsModel_ViewWithNilForm(t *testing.T) {
	model := SettingsModel{}

	view := model.View()
	if view.Content != "Unable to render settings form" {
		t.Fatalf("expected view content %q, got %q", "Unable to render settings form", view.Content)
	}
}

func TestSettingsModel_ViewIncludesError(t *testing.T) {
	model := NewSettingsModel(nil, "", 80, 24)
	model.err = config.Save(nil, "")

	view := model.View()
	if !strings.Contains(view.Content, "Error: config cannot be nil") {
		t.Fatalf("expected error text in view, got %q", view.Content)
	}
}
