package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// WelcomeSelectMsg is sent when the user selects a menu item.
type WelcomeSelectMsg struct {
	Choice string // "notes", "new", "settings", "quit"
}

type WelcomeModel struct {
	width   int
	height  int
	cursor  int
	choices []welcomeChoice
	version string
}

type welcomeChoice struct {
	label string
	key   string
}

func NewWelcomeModel(width, height int, version string) WelcomeModel {
	if version == "" {
		version = "dev"
	}

	return WelcomeModel{
		width:   width,
		height:  height,
		version: version,
		choices: []welcomeChoice{
			{label: "📋 Explore Notes", key: "notes"},
			{label: "✏️  New Meeting", key: "new"},
			{label: "⚙️  Settings", key: "settings"},
			{label: "🚪 Quit", key: "quit"},
		},
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
	case tea.KeyPressMsg:
		switch typedMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return WelcomeSelectMsg{Choice: m.choices[m.cursor].key}
			}
		}
	}

	return m, nil
}

const recapLogo = `
 ____
|  _ \ ___  ___ __ _ _ __
| |_) / _ \/ __/ _` + "`" + ` | '_ \
|  _ <  __/ (_| (_| | |_) |
|_| \_\___|\___\__,_| .__/
                     |_|    `

func (m WelcomeModel) View() tea.View {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var b strings.Builder

	// Logo
	b.WriteString(titleStyle.Render(recapLogo))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("  Meeting notes with AI-powered structuring"))
	b.WriteString("\n")

	versionLine := "  v" + m.version
	if strings.HasPrefix(m.version, "v") {
		versionLine = "  " + m.version
	}
	b.WriteString(subtitleStyle.Render(versionLine))
	b.WriteString("\n\n")

	// Menu
	for i, choice := range m.choices {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		b.WriteString(cursor + style.Render(choice.label) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("  ↑/↓ navigate • enter select"))

	// Center the content
	content := b.String()
	contentWidth := lipgloss.Width(content)
	contentHeight := strings.Count(content, "\n") + 1

	padLeft := 0
	if m.width > contentWidth {
		padLeft = (m.width - contentWidth) / 2
	}
	padTop := 0
	if m.height > contentHeight {
		padTop = (m.height - contentHeight) / 3
	}

	wrapper := lipgloss.NewStyle().
		PaddingLeft(padLeft).
		PaddingTop(padTop)

	return tea.NewView(wrapper.Render(content))
}
