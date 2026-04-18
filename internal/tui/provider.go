package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ProviderItem struct {
	name        string
	displayName string
	description string
	current     bool
}

func (i ProviderItem) FilterValue() string {
	return i.displayName
}

func (i ProviderItem) Title() string {
	if i.current {
		return "✓ " + i.displayName
	}

	return "  " + i.displayName
}

func (i ProviderItem) Description() string {
	return i.description
}

type ProviderSelectedMsg struct {
	ProviderName string
}

type ProviderModel struct {
	list   list.Model
	width  int
	height int
}

func NewProviderModel(currentProvider string, width, height int) ProviderModel {
	items := []list.Item{
		ProviderItem{
			name:        "github_models",
			displayName: "GitHub Models",
			description: "GitHub-hosted AI models (requires gh auth)",
			current:     currentProvider == "github_models",
		},
		ProviderItem{
			name:        "openrouter",
			displayName: "OpenRouter",
			description: "OpenRouter API (requires API key)",
			current:     currentProvider == "openrouter",
		},
		ProviderItem{
			name:        "lm_studio",
			displayName: "LM Studio",
			description: "Local LM Studio server",
			current:     currentProvider == "lm_studio",
		},
	}

	l := list.New(items, list.NewDefaultDelegate(), width, height-2)
	l.Title = lipgloss.NewStyle().Render("Select AI Provider")
	l.SetFilteringEnabled(false)

	return ProviderModel{
		list:   l,
		width:  width,
		height: height,
	}
}

func (m ProviderModel) Init() tea.Cmd {
	return nil
}

func (m ProviderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.KeyPressMsg:
		switch typedMsg.String() {
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(ProviderItem)
			if !ok {
				return m, nil
			}

			return m, func() tea.Msg {
				return ProviderSelectedMsg{ProviderName: selectedItem.name}
			}
		case "esc":
			return m, func() tea.Msg {
				return NavigateMsg{Screen: ScreenEditor}
			}
		}
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.list.SetSize(m.width, m.height-2)
		return m, nil
	}

	updatedList, cmd := m.list.Update(msg)
	m.list = updatedList
	return m, cmd
}

func (m ProviderModel) View() tea.View {
	return tea.NewView(m.list.View())
}
