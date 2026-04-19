package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ShowLanguageModalMsg struct {
	CurrentLanguage string
}

type LanguageSelectModel struct {
	options []string
	labels  []string
	current string
	cursor  int
}

func NewLanguageSelectModel(current string, configuredCodes []string) LanguageSelectModel {
	options := configuredCodes
	if len(options) == 0 {
		options = []string{"en"}
	}

	labels := make([]string, len(options))
	for i, option := range options {
		labels[i] = strings.ToUpper(option)
	}

	cursor := 0
	for i, option := range options {
		if option == current {
			cursor = i
			break
		}
	}

	return LanguageSelectModel{
		options: options,
		labels:  labels,
		current: current,
		cursor:  cursor,
	}
}

func (m *LanguageSelectModel) Init() tea.Cmd {
	return nil
}

func (m *LanguageSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		m.cursor--
		if m.cursor < 0 {
			m.cursor = len(m.options) - 1
		}
		return m, nil
	case "down", "j":
		m.cursor++
		if m.cursor >= len(m.options) {
			m.cursor = 0
		}
		return m, nil
	case "enter":
		selectedLanguage := m.options[m.cursor]
		if selectedLanguage == m.current {
			return m, func() tea.Msg { return DismissModalMsg{} }
		}

		return m, func() tea.Msg { return LanguageChangedMsg{Language: selectedLanguage} }
	default:
		return m, nil
	}
}

func (m *LanguageSelectModel) View() tea.View {
	faintStyle := lipgloss.NewStyle().Faint(true)
	lines := make([]string, 0, len(m.options))

	for i := range m.options {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		suffix := ""
		if m.options[i] == m.current {
			suffix = " *"
		}

		line := prefix + m.labels[i] + suffix
		if i != m.cursor {
			line = faintStyle.Render(line)
		}

		lines = append(lines, line)
	}

	return tea.NewView(strings.Join(lines, "\n"))
}
