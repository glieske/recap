package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type HelpModel struct {
	width  int
	height int
	screen Screen
}

func NewHelpModel() HelpModel {
	return HelpModel{}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
	}

	return m, nil
}

func (m HelpModel) View() tea.View {
	titleStyle := lipgloss.NewStyle().Bold(true)
	sectionHeaderStyle := lipgloss.NewStyle().Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	title := titleStyle.Render("⌨ Keybindings")

	type section struct {
		header   string
		bindings [][2]string
	}

	allSections := []section{
		{
			header: "Global",
			bindings: [][2]string{
				{"q / Ctrl+C", "Quit"},
				{"?", "Toggle help"},
				{"Esc", "Back / close"},
				{"Ctrl+,", "Open settings"},
			},
		},
		{
			header: "Welcome Screen",
			bindings: [][2]string{
				{"↑/↓ / j/k", "Navigate menu"},
				{"Enter", "Select item"},
			},
		},
		{
			header: "Meeting List",
			bindings: [][2]string{
				{"Enter", "Open selected meeting"},
				{"n", "New meeting"},
				{"f", "Cycle project filter"},
				{"t", "Cycle tag filter"},
				{"d", "Delete meeting"},
				{"/", "Search"},
			},
		},
		{
			header: "Editor",
			bindings: [][2]string{
				{"Ctrl+S", "Save"},
				{"Ctrl+T", "Insert timestamp [HH:MM]"},
				{"Ctrl+A", "AI structure notes"},
				{"Ctrl+E", "Generate email summary"},
				{"Ctrl+P", "Toggle split view"},
				{"Ctrl+O", "Select AI provider"},
				{"Ctrl+,", "Open settings"},
			},
		},
		{
			header: "Split Pane",
			bindings: [][2]string{
				{"Tab", "Switch focus between panes"},
				{"Ctrl+S", "Save (left: raw notes, right: summary)"},
				{"Esc", "Collapse split view"},
			},
		},
		{
			header: "Email",
			bindings: [][2]string{
				{"c", "Copy to clipboard"},
				{"r", "Regenerate email"},
				{"l", "Cycle email language (en → pl → no)"},
			},
		},
		{
			header: "Confirmation Dialogs",
			bindings: [][2]string{
				{"y / Enter", "Confirm action"},
				{"n / Esc", "Cancel action"},
				{"←/→/Tab", "Switch between Yes/No"},
			},
		},
	}

	// visibleHeaders determines which sections are shown for each screen.
	// "Global" is always included.
	var visibleHeaders map[string]bool
	switch m.screen {
	case ScreenWelcome:
		visibleHeaders = map[string]bool{"Global": true, "Welcome Screen": true}
	case ScreenMeetingList:
		visibleHeaders = map[string]bool{"Global": true, "Meeting List": true, "Confirmation Dialogs": true}
	case ScreenEditor:
		visibleHeaders = map[string]bool{"Global": true, "Editor": true, "Split Pane": true, "Confirmation Dialogs": true}
	case ScreenEmail:
		visibleHeaders = map[string]bool{"Global": true, "Email": true}
	default:
		visibleHeaders = map[string]bool{"Global": true}
	}

	visible := make([]section, 0, len(allSections))
	for _, section := range allSections {
		if visibleHeaders[section.header] {
			visible = append(visible, section)
		}
	}

	lines := make([]string, 0, 32)
	lines = append(lines, title, "")

	for sectionIndex, section := range visible {
		lines = append(lines, sectionHeaderStyle.Render(section.header))
		for _, binding := range section.bindings {
			lines = append(lines, "  "+keyStyle.Render(binding[0])+" — "+binding[1])
		}

		if sectionIndex < len(visible)-1 {
			lines = append(lines, "")
		}
	}

	content := strings.Join(lines, "\n")
	return tea.NewView(content)
}
