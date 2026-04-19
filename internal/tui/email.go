package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
)

const emailPlaceholder = "No email generated yet. Press Ctrl+E in the editor to generate."

type CopyToClipboardMsg struct{}

type ClipboardDoneMsg struct{}

type ClipboardErrMsg struct {
	Err error
}

type ClearStatusMsg struct{}

type RegenerateEmailMsg struct{}

type LanguageChangedMsg struct {
	Language string
}

type EmailContentMsg struct {
	Subject string
	Body    string
}

type EmailModel struct {
	viewport     viewport.Model
	subject      string
	body         string
	language     string
	version      string
	width        int
	height       int
	clipboardErr error
	statusMsg    string
}

func NewEmailModel(subject, body string, width, height int, language string, version string) EmailModel {
	if language == "" {
		language = "pl"
	}

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(emailViewportHeight(height)))
	vp.SetContent(formatEmailContent(subject, body))

	return EmailModel{
		viewport: vp,
		subject:  subject,
		body:     body,
		language: language,
		version:  version,
		width:    width,
		height:   height,
	}
}

func nextEmailLanguage(current string) string {
	switch current {
	case "en":
		return "pl"
	case "pl":
		return "no"
	case "no":
		return "en"
	default:
		return "pl"
	}
}

func emailLanguageDisplayName(lang string) string {
	switch lang {
	case "en":
		return "EN"
	case "pl":
		return "PL"
	case "no":
		return "NO"
	default:
		return "PL"
	}
}

func (m EmailModel) Init() tea.Cmd {
	return nil
}

func (m EmailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.viewport.SetWidth(m.width)
		m.viewport.SetHeight(emailViewportHeight(m.height))
		return m, nil
	case EmailContentMsg:
		m.subject = typedMsg.Subject
		m.body = typedMsg.Body
		m.viewport.SetContent(formatEmailContent(m.subject, m.body))
		return m, nil
	case tea.KeyPressMsg:
		switch typedMsg.String() {
		case "l":
			return m, func() tea.Msg { return ShowLanguageModalMsg{CurrentLanguage: m.language} }
		case "c":
			if m.subject == "" && m.body == "" {
				m.statusMsg = "Nothing to copy"
				return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return ClearStatusMsg{} })
			}

			return m, func() tea.Msg {
				copyText := m.subject + "\n\n" + m.body
				if err := clipboard.WriteAll(copyText); err != nil {
					return ClipboardErrMsg{Err: err}
				}

				return ClipboardDoneMsg{}
			}
		case "r":
			return m, func() tea.Msg { return RegenerateEmailMsg{} }
		case "esc":
			return m, func() tea.Msg { return NavigateMsg{Screen: ScreenMeetingList} }
		default:
			updatedViewport, cmd := m.viewport.Update(msg)
			m.viewport = updatedViewport
			return m, cmd
		}
	case ClipboardDoneMsg:
		m.clipboardErr = nil
		m.statusMsg = "Copied to clipboard!"
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return ClearStatusMsg{} })
	case ClipboardErrMsg:
		m.clipboardErr = typedMsg.Err
		m.statusMsg = "Clipboard unavailable: " + typedMsg.Err.Error()
		return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return ClearStatusMsg{} })
	case ClearStatusMsg:
		m.statusMsg = ""
		return m, nil
	default:
		updatedViewport, cmd := m.viewport.Update(msg)
		m.viewport = updatedViewport
		return m, cmd
	}
}

func (m EmailModel) View() tea.View {
	headerStyle := lipgloss.NewStyle().Faint(true)
	footerStyle := lipgloss.NewStyle().Faint(true)

	header := headerStyle.Render("── Email Summary ──")
	left := fmt.Sprintf("c copy • r regenerate • l lang:%s • Esc back", emailLanguageDisplayName(m.language))

	if m.statusMsg != "" {
		left = m.statusMsg
	}

	right := ""
	if m.version != "" {
		right = "recap " + m.version
	}

	footerText := composeStatusLine(left, "", right, m.width)

	footer := footerStyle.Render(footerText)

	return tea.NewView(fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer))
}

func formatEmailContent(subject, body string) string {
	if subject == "" && body == "" {
		return emailPlaceholder
	}

	return fmt.Sprintf("Subject: %s\n\n%s", subject, body)
}

func emailViewportHeight(totalHeight int) int {
	viewportHeight := totalHeight - 2
	if viewportHeight < 1 {
		return 1
	}

	return viewportHeight
}
