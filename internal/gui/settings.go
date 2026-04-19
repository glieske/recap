//go:build gui

package gui

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/languages"
)

func NewSettingsScreen(cfg *config.Config, configPath string, win fyne.Window) fyne.CanvasObject {
	providerSelect := widget.NewSelect([]string{"github_models", "openrouter", "lm_studio"}, nil)
	providerSelect.SetSelected(cfg.AIProvider)

	githubModelEntry := widget.NewEntry()
	githubModelEntry.SetText(cfg.GitHubModel)

	openRouterModelEntry := widget.NewEntry()
	openRouterModelEntry.SetText(cfg.OpenRouterModel)

	openRouterAPIKeyEntry := widget.NewPasswordEntry()
	openRouterAPIKeyEntry.SetText(cfg.OpenRouterAPIKey)

	lmStudioURLEntry := widget.NewEntry()
	lmStudioURLEntry.SetText(cfg.LMStudioURL)

	lmStudioModelEntry := widget.NewEntry()
	lmStudioModelEntry.SetText(cfg.LMStudioModel)

	providerSpecificStack := container.NewStack()
	updateProviderSpecificFields := func(providerName string) {
		switch providerName {
		case "openrouter":
			providerSpecificStack.Objects = []fyne.CanvasObject{
				widget.NewForm(
					widget.NewFormItem("OpenRouter Model", openRouterModelEntry),
					widget.NewFormItem("OpenRouter API Key", openRouterAPIKeyEntry),
				),
			}
		case "lm_studio":
			providerSpecificStack.Objects = []fyne.CanvasObject{
				widget.NewForm(
					widget.NewFormItem("LM Studio URL", lmStudioURLEntry),
					widget.NewFormItem("LM Studio Model", lmStudioModelEntry),
				),
			}
		default:
			providerSpecificStack.Objects = []fyne.CanvasObject{
				widget.NewForm(
					widget.NewFormItem("GitHub Model", githubModelEntry),
				),
			}
		}
		providerSpecificStack.Refresh()
	}

	providerSelect.OnChanged = updateProviderSpecificFields
	if providerSelect.Selected == "" {
		providerSelect.SetSelected("github_models")
	}
	updateProviderSpecificFields(providerSelect.Selected)

	displayToCode := make(map[string]string, len(languages.AllLanguages))
	allDisplayNames := make([]string, 0, len(languages.AllLanguages))
	for _, language := range languages.AllLanguages {
		displayToCode[language.Name] = language.Code
		allDisplayNames = append(allDisplayNames, language.Name)
	}

	selectedNames := make([]string, 0, len(cfg.EmailLanguages))
	for _, code := range cfg.EmailLanguages {
		selectedNames = append(selectedNames, languages.DisplayName(code))
	}

	languageCheckGroup := widget.NewCheckGroup(allDisplayNames, nil)
	languageCheckGroup.SetSelected(selectedNames)

	notesDirEntry := widget.NewEntry()
	notesDirEntry.SetText(cfg.NotesDir)

	saveButton := widget.NewButton("Save Settings", func() {
		notesDirectory := strings.TrimSpace(notesDirEntry.Text)
		if notesDirectory == "" {
			dialog.ShowError(fmt.Errorf("notes directory must not be empty"), win)
			return
		}

		selectedProvider := providerSelect.Selected
		if selectedProvider == "" {
			selectedProvider = "github_models"
		}

		lmStudioURL := strings.TrimSpace(lmStudioURLEntry.Text)
		if selectedProvider == "lm_studio" && lmStudioURL != "" && !strings.HasPrefix(lmStudioURL, "http://") && !strings.HasPrefix(lmStudioURL, "https://") {
			dialog.ShowError(fmt.Errorf("LM Studio URL must start with http:// or https://"), win)
			return
		}

		selectedDisplayNames := languageCheckGroup.Selected
		if len(selectedDisplayNames) < languages.MinSelected {
			dialog.ShowError(fmt.Errorf("select at least %d language", languages.MinSelected), win)
			return
		}
		if len(selectedDisplayNames) > languages.MaxSelected {
			dialog.ShowError(fmt.Errorf("select at most %d languages", languages.MaxSelected), win)
			return
		}

		selectedCodes := make([]string, 0, len(selectedDisplayNames))
		for _, name := range selectedDisplayNames {
			if code, ok := displayToCode[name]; ok {
				selectedCodes = append(selectedCodes, code)
			}
		}

		newCfg := &config.Config{
			NotesDir:         notesDirectory,
			AIProvider:       selectedProvider,
			GitHubModel:      strings.TrimSpace(githubModelEntry.Text),
			OpenRouterModel:  strings.TrimSpace(openRouterModelEntry.Text),
			OpenRouterAPIKey: strings.TrimSpace(openRouterAPIKeyEntry.Text),
			LMStudioURL:      lmStudioURL,
			LMStudioModel:    strings.TrimSpace(lmStudioModelEntry.Text),
			EmailLanguages:   selectedCodes,
		}

		if err := config.Save(newCfg, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "error saving settings: %v\n", err)
			dialog.ShowError(err, win)
			return
		}

		*cfg = *newCfg
		dialog.ShowInformation("Settings Saved", "Settings were saved successfully.", win)
	})
	saveButton.Importance = widget.HighImportance

	providerHeader := widget.NewLabel("AI Provider")
	providerHeader.TextStyle = fyne.TextStyle{Bold: true}

	emailHeader := widget.NewLabel("Email Languages")
	emailHeader.TextStyle = fyne.TextStyle{Bold: true}

	notesDirectoryHeader := widget.NewLabel("Notes Directory")
	notesDirectoryHeader.TextStyle = fyne.TextStyle{Bold: true}

	providerForm := widget.NewForm(widget.NewFormItem("Provider", providerSelect))
	languageForm := widget.NewForm(widget.NewFormItem("Languages", languageCheckGroup))
	notesDirectoryForm := widget.NewForm(widget.NewFormItem("Directory", notesDirEntry))

	return container.NewVScroll(
		container.NewVBox(
			providerHeader,
			providerForm,
			providerSpecificStack,
			widget.NewSeparator(),
			emailHeader,
			languageForm,
			widget.NewSeparator(),
			notesDirectoryHeader,
			notesDirectoryForm,
			widget.NewSeparator(),
			saveButton,
		),
	)
}
