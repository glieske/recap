//go:build gui

package gui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/config"
)

func TestSettingsRendersProviderSelect(t *testing.T) {
	a := fyneTest.NewApp()
	defer a.Quit()

	w := a.NewWindow("settings-provider-default")
	cfg := &config.Config{
		AIProvider:     "",
		GitHubModel:    "gpt-4o",
		LMStudioURL:    "http://localhost:1234/v1",
		EmailLanguages: []string{"en"},
		NotesDir:       t.TempDir(),
	}

	screen := NewSettingsScreen(cfg, "", w)
	w.SetContent(screen)

	providerSelect := findSettingsSelectByOptions(screen, []string{"github_models", "openrouter", "lm_studio"})
	if providerSelect == nil {
		t.Fatal("expected provider select with github_models/openrouter/lm_studio options")
	}

	if got, want := providerSelect.Selected, "github_models"; got != want {
		t.Fatalf("expected provider default %q, got %q", want, got)
	}
}

func TestSettingsProviderSwapToOpenRouter(t *testing.T) {
	a := fyneTest.NewApp()
	defer a.Quit()

	w := a.NewWindow("settings-provider-openrouter")
	cfg := &config.Config{
		AIProvider:       "github_models",
		GitHubModel:      "gpt-4o",
		OpenRouterModel:  "google/gemini-flash-1.5",
		OpenRouterAPIKey: "api-key",
		LMStudioURL:      "http://localhost:1234/v1",
		EmailLanguages:   []string{"en"},
		NotesDir:         t.TempDir(),
	}

	screen := NewSettingsScreen(cfg, "", w)
	w.SetContent(screen)

	providerSelect := findSettingsSelectByOptions(screen, []string{"github_models", "openrouter", "lm_studio"})
	if providerSelect == nil {
		t.Fatal("expected provider select")
	}

	providerSelect.SetSelected("openrouter")

	if !settingsHasFormItemLabel(screen, "OpenRouter Model") {
		t.Fatal("expected OpenRouter Model field after selecting openrouter")
	}
	if !settingsHasFormItemLabel(screen, "OpenRouter API Key") {
		t.Fatal("expected OpenRouter API Key field after selecting openrouter")
	}
	if settingsHasFormItemLabel(screen, "GitHub Model") {
		t.Fatal("did not expect GitHub Model field after selecting openrouter")
	}
}

func TestSettingsProviderSwapToLMStudio(t *testing.T) {
	a := fyneTest.NewApp()
	defer a.Quit()

	w := a.NewWindow("settings-provider-lmstudio")
	cfg := &config.Config{
		AIProvider:     "github_models",
		GitHubModel:    "gpt-4o",
		LMStudioURL:    "http://localhost:1234/v1",
		LMStudioModel:  "llama3",
		EmailLanguages: []string{"en"},
		NotesDir:       t.TempDir(),
	}

	screen := NewSettingsScreen(cfg, "", w)
	w.SetContent(screen)

	providerSelect := findSettingsSelectByOptions(screen, []string{"github_models", "openrouter", "lm_studio"})
	if providerSelect == nil {
		t.Fatal("expected provider select")
	}

	providerSelect.SetSelected("lm_studio")

	if !settingsHasFormItemLabel(screen, "LM Studio URL") {
		t.Fatal("expected LM Studio URL field after selecting lm_studio")
	}
	if !settingsHasFormItemLabel(screen, "LM Studio Model") {
		t.Fatal("expected LM Studio Model field after selecting lm_studio")
	}
	if settingsHasFormItemLabel(screen, "OpenRouter Model") {
		t.Fatal("did not expect OpenRouter Model field after selecting lm_studio")
	}
}

func TestSettingsEmailLanguageMapping(t *testing.T) {
	a := fyneTest.NewApp()
	defer a.Quit()

	w := a.NewWindow("settings-language")
	cfg := &config.Config{
		AIProvider:     "github_models",
		GitHubModel:    "gpt-4o",
		LMStudioURL:    "http://localhost:1234/v1",
		EmailLanguages: []string{"pl"},
		NotesDir:       t.TempDir(),
	}

	screen := NewSettingsScreen(cfg, "", w)
	w.SetContent(screen)

	languageCheckGroup := findSettingsCheckGroup(screen)
	if languageCheckGroup == nil {
		t.Fatal("expected language check group")
	}

	selected := languageCheckGroup.Selected
	if len(selected) != 1 || selected[0] != "Polish" {
		t.Fatalf("expected Selected=[Polish] for cfg.EmailLanguages=[pl], got %v", selected)
	}
}

func TestSettingsSaveUpdatesConfig(t *testing.T) {
	a := fyneTest.NewApp()
	defer a.Quit()

	w := a.NewWindow("settings-save")
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	originalNotesDir := filepath.Join(tempDir, "notes-original")
	cfg := &config.Config{
		AIProvider:       "github_models",
		GitHubModel:      "gpt-4o",
		OpenRouterModel:  "google/gemini-flash-1.5",
		OpenRouterAPIKey: "",
		LMStudioURL:      "http://localhost:1234/v1",
		LMStudioModel:    "",
		EmailLanguages:   []string{"en"},
		NotesDir:         originalNotesDir,
	}

	screen := NewSettingsScreen(cfg, configPath, w)
	w.SetContent(screen)

	providerSelect := findSettingsSelectByOptions(screen, []string{"github_models", "openrouter", "lm_studio"})
	if providerSelect == nil {
		t.Fatal("expected provider select")
	}
	languageCheckGroup := findSettingsCheckGroup(screen)
	if languageCheckGroup == nil {
		t.Fatal("expected language check group")
	}

	notesDirEntry := findSettingsEntryByFormLabel(screen, "Directory")
	if notesDirEntry == nil {
		t.Fatal("expected notes directory entry")
	}

	saveButton := findSettingsButtonByText(screen, "Save Settings")
	if saveButton == nil {
		t.Fatal("expected Save Settings button")
	}

	notesDirEntry.SetText("")
	fyneTest.Tap(saveButton)

	if got, want := cfg.NotesDir, originalNotesDir; got != want {
		t.Fatalf("expected cfg.NotesDir to remain %q after validation failure, got %q", want, got)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config file to not be written on validation failure, stat error=%v", err)
	}

	providerSelect.SetSelected("openrouter")
	openRouterModelEntry := findSettingsEntryByFormLabel(screen, "OpenRouter Model")
	if openRouterModelEntry == nil {
		t.Fatal("expected OpenRouter Model entry after selecting openrouter")
	}
	openRouterAPIKeyEntry := findSettingsEntryByFormLabel(screen, "OpenRouter API Key")
	if openRouterAPIKeyEntry == nil {
		t.Fatal("expected OpenRouter API Key entry after selecting openrouter")
	}

	updatedNotesDir := filepath.Join(tempDir, "notes-updated")
	notesDirEntry.SetText(updatedNotesDir)
	openRouterModelEntry.SetText("openrouter/new-model")
	openRouterAPIKeyEntry.SetText("test-api-key")
	languageCheckGroup.SetSelected([]string{"Norwegian"})
	fyneTest.Tap(saveButton)

	if got, want := cfg.AIProvider, "openrouter"; got != want {
		t.Fatalf("expected cfg.AIProvider %q, got %q", want, got)
	}
	if got, want := cfg.NotesDir, updatedNotesDir; got != want {
		t.Fatalf("expected cfg.NotesDir %q, got %q", want, got)
	}
	if got, want := cfg.OpenRouterModel, "openrouter/new-model"; got != want {
		t.Fatalf("expected cfg.OpenRouterModel %q, got %q", want, got)
	}
	if got, want := cfg.OpenRouterAPIKey, "test-api-key"; got != want {
		t.Fatalf("expected cfg.OpenRouterAPIKey %q, got %q", want, got)
	}
	if len(cfg.EmailLanguages) != 1 || cfg.EmailLanguages[0] != "no" {
		t.Fatalf("expected cfg.EmailLanguages [no] from Norwegian selection, got %v", cfg.EmailLanguages)
	}

	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("expected config.Load to read saved file: %v", err)
	}
	if got, want := loadedCfg.AIProvider, "openrouter"; got != want {
		t.Fatalf("expected saved ai_provider %q, got %q", want, got)
	}
	if len(loadedCfg.EmailLanguages) != 1 || loadedCfg.EmailLanguages[0] != "no" {
		t.Fatalf("expected saved email_languages [no], got %v", loadedCfg.EmailLanguages)
	}
	if got, want := loadedCfg.NotesDir, updatedNotesDir; got != want {
		t.Fatalf("expected saved notes_dir %q, got %q", want, got)
	}
}

func findSettingsSelectByOptions(root fyne.CanvasObject, options []string) *widget.Select {
	var found *widget.Select
	walkSettingsObjects(root, func(obj fyne.CanvasObject) {
		if found != nil {
			return
		}
		sel, ok := obj.(*widget.Select)
		if !ok {
			return
		}
		if reflect.DeepEqual(sel.Options, options) {
			found = sel
		}
	})

	return found
}

func findSettingsCheckGroup(root fyne.CanvasObject) *widget.CheckGroup {
	var found *widget.CheckGroup
	walkSettingsObjects(root, func(obj fyne.CanvasObject) {
		if found != nil {
			return
		}
		cg, ok := obj.(*widget.CheckGroup)
		if ok {
			found = cg
		}
	})

	return found
}

func settingsHasFormItemLabel(root fyne.CanvasObject, label string) bool {
	return findSettingsFormItemWidget(root, label) != nil
}

func findSettingsEntryByFormLabel(root fyne.CanvasObject, label string) *widget.Entry {
	widgetObj := findSettingsFormItemWidget(root, label)
	if widgetObj == nil {
		return nil
	}
	entry, _ := widgetObj.(*widget.Entry)
	return entry
}

func findSettingsFormItemWidget(root fyne.CanvasObject, label string) fyne.CanvasObject {
	var found fyne.CanvasObject
	walkSettingsObjects(root, func(obj fyne.CanvasObject) {
		if found != nil {
			return
		}
		form, ok := obj.(*widget.Form)
		if !ok {
			return
		}
		for _, item := range form.Items {
			if item != nil && item.Text == label {
				found = item.Widget
				return
			}
		}
	})

	return found
}

func findSettingsButtonByText(root fyne.CanvasObject, text string) *widget.Button {
	var found *widget.Button
	walkSettingsObjects(root, func(obj fyne.CanvasObject) {
		if found != nil {
			return
		}
		button, ok := obj.(*widget.Button)
		if !ok {
			return
		}
		if button.Text == text {
			found = button
		}
	})

	return found
}

func walkSettingsObjects(obj fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if obj == nil {
		return
	}

	visit(obj)

	if scroll, ok := obj.(*container.Scroll); ok {
		walkSettingsObjects(scroll.Content, visit)
		return
	}

	if form, ok := obj.(*widget.Form); ok {
		for _, item := range form.Items {
			if item == nil {
				continue
			}
			walkSettingsObjects(item.Widget, visit)
		}
		return
	}

	if containerObj, ok := obj.(*fyne.Container); ok {
		for _, child := range containerObj.Objects {
			walkSettingsObjects(child, visit)
		}
	}
}
