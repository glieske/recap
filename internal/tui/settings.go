package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/glieske/recap/internal/config"
)

type SettingsUpdatedMsg struct {
	Config *config.Config
}

type settingsFormValues struct {
	provider         string
	githubModel      string
	openrouterModel  string
	openrouterAPIKey string
	lmStudioURL      string
	lmStudioModel    string
	emailLanguage    string
	notesDir         string
}

type SettingsModel struct {
	form       *huh.Form
	cfg        *config.Config
	configPath string
	submitted  bool
	cancelled  bool
	width      int
	height     int
	values     *settingsFormValues
	err        error
}

func NewSettingsModel(cfg *config.Config, configPath string, width, height int) SettingsModel {
	values := &settingsFormValues{}
	if cfg != nil {
		values.provider = cfg.AIProvider
		values.githubModel = cfg.GitHubModel
		values.openrouterModel = cfg.OpenRouterModel
		values.openrouterAPIKey = cfg.OpenRouterAPIKey
		values.lmStudioURL = cfg.LMStudioURL
		values.lmStudioModel = cfg.LMStudioModel
		values.emailLanguage = cfg.EmailLanguage
		values.notesDir = cfg.NotesDir
	}

	if values.provider != "github_models" && values.provider != "openrouter" && values.provider != "lm_studio" {
		values.provider = "github_models"
	}

	if values.emailLanguage != "en" && values.emailLanguage != "pl" && values.emailLanguage != "no" {
		values.emailLanguage = "en"
	}

	model := SettingsModel{
		cfg:        cfg,
		configPath: configPath,
		width:      width,
		height:     height,
		values:     values,
	}

	model.form = buildSettingsForm(values)

	return model
}

func (m SettingsModel) Init() tea.Cmd {
	if m.form == nil {
		return nil
	}

	return m.form.Init()
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
	}

	if m.form == nil {
		return m, nil
	}

	updatedForm, formCmd := m.form.Update(msg)
	if castForm, ok := updatedForm.(*huh.Form); ok {
		m.form = castForm
	}

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true

		newCfg, validationErr := m.configFromValues()
		if validationErr != nil {
			m.err = validationErr
			return m, nil
		}

		if saveErr := config.Save(newCfg, m.configPath); saveErr != nil {
			m.err = fmt.Errorf("save settings: %w", saveErr)
			return m, nil
		}

		if m.cfg != nil {
			*m.cfg = *newCfg
		}

		m.err = nil
		return m, func() tea.Msg {
			return SettingsUpdatedMsg{Config: newCfg}
		}
	}

	if m.form.State == huh.StateAborted && !m.cancelled {
		m.cancelled = true
		return m, func() tea.Msg {
			return DismissModalMsg{}
		}
	}

	return m, formCmd
}

func (m SettingsModel) View() tea.View {
	if m.form == nil {
		return tea.NewView("Unable to render settings form")
	}

	view := m.form.View()
	if m.err != nil {
		view += "\n\nError: " + m.err.Error()
	}

	return tea.NewView(view)
}

func (m SettingsModel) configFromValues() (*config.Config, error) {
	providerValue := strings.TrimSpace(m.values.provider)
	if providerValue != "github_models" && providerValue != "openrouter" && providerValue != "lm_studio" {
		return nil, fmt.Errorf("invalid AI provider: %s", providerValue)
	}

	notesDirValue := strings.TrimSpace(m.values.notesDir)
	if notesDirValue == "" {
		return nil, fmt.Errorf("notes directory is required")
	}

	lmStudioURLValue := strings.TrimSpace(m.values.lmStudioURL)
	if lmStudioURLValue != "" && !strings.HasPrefix(lmStudioURLValue, "http://") && !strings.HasPrefix(lmStudioURLValue, "https://") {
		return nil, fmt.Errorf("LM Studio URL must start with http:// or https://")
	}

	return &config.Config{
		NotesDir:         notesDirValue,
		AIProvider:       providerValue,
		GitHubModel:      strings.TrimSpace(m.values.githubModel),
		OpenRouterModel:  strings.TrimSpace(m.values.openrouterModel),
		OpenRouterAPIKey: strings.TrimSpace(m.values.openrouterAPIKey),
		LMStudioURL:      lmStudioURLValue,
		LMStudioModel:    strings.TrimSpace(m.values.lmStudioModel),
		EmailLanguage:    strings.TrimSpace(m.values.emailLanguage),
	}, nil
}

func buildSettingsForm(values *settingsFormValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI Provider").
				Options(
					huh.NewOption("GitHub Models", "github_models"),
					huh.NewOption("OpenRouter", "openrouter"),
					huh.NewOption("LM Studio", "lm_studio"),
				).
				Value(&values.provider),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub Model Name").
				Value(&values.githubModel),
		).WithHideFunc(func() bool { return values.provider != "github_models" }),
		huh.NewGroup(
			huh.NewInput().
				Title("OpenRouter Model").
				Value(&values.openrouterModel),
			huh.NewInput().
				Title("OpenRouter API Key").
				EchoMode(huh.EchoModePassword).
				Value(&values.openrouterAPIKey),
		).WithHideFunc(func() bool { return values.provider != "openrouter" }),
		huh.NewGroup(
			huh.NewInput().
				Title("LM Studio URL").
				Value(&values.lmStudioURL).
				Validate(func(inputText string) error {
					trimmedValue := strings.TrimSpace(inputText)
					if trimmedValue == "" {
						return nil
					}

					if !strings.HasPrefix(trimmedValue, "http://") && !strings.HasPrefix(trimmedValue, "https://") {
						return fmt.Errorf("URL must start with http:// or https://")
					}

					return nil
				}),
			huh.NewInput().
				Title("LM Studio Model").
				Value(&values.lmStudioModel),
		).WithHideFunc(func() bool { return values.provider != "lm_studio" }),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Email Language").
				Options(
					huh.NewOption("English", "en"),
					huh.NewOption("Polish", "pl"),
					huh.NewOption("Norwegian", "no"),
				).
				Value(&values.emailLanguage),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Notes Directory").
				Value(&values.notesDir).
				Validate(func(inputText string) error {
					if strings.TrimSpace(inputText) == "" {
						return fmt.Errorf("notes directory is required")
					}

					return nil
				}),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeDracula))
}
