package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ConfirmResultMsg struct {
	Confirmed bool
	Action    string
}

type ConfirmModel struct {
	question  string
	action    string
	confirmed bool
	focused   int
}

func NewConfirmModel(question string, action string) ConfirmModel {
	return ConfirmModel{
		question: question,
		action:   action,
		focused:  0,
	}
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if keyMsg.Text == "y" {
		m.confirmed = true
		return m, func() tea.Msg {
			return ConfirmResultMsg{Confirmed: true, Action: m.action}
		}
	}

	if keyMsg.Text == "n" {
		m.confirmed = false
		return m, func() tea.Msg {
			return ConfirmResultMsg{Confirmed: false, Action: m.action}
		}
	}

	switch keyMsg.String() {
	case "enter":
		m.confirmed = m.focused == 0
		return m, func() tea.Msg {
			return ConfirmResultMsg{Confirmed: m.confirmed, Action: m.action}
		}
	case "left", "right", "tab":
		m.focused = 1 - m.focused
		return m, nil
	default:
		return m, nil
	}
}

func (m ConfirmModel) View() tea.View {
	focusedButtonStyle := lipgloss.NewStyle().Bold(true)

	yesButton := "[ Yes ]"
	noButton := "[ No ]"
	if m.focused == 0 {
		yesButton = focusedButtonStyle.Render(yesButton)
	} else {
		noButton = focusedButtonStyle.Render(noButton)
	}

	view := m.question + "\n\n" + yesButton + " " + noButton
	return tea.NewView(view)
}
