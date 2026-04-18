package tui

import (
	"errors"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/glieske/recap/internal/storage"
)

type SummarySaveDoneMsg struct{}

type SummarySaveErrMsg struct {
	Err error
}

type SummaryModel struct {
	textarea                textarea.Model
	store                   *storage.Store
	project                 string
	meetingID               string
	width                   int
	height                  int
	dirty                   bool
	overwriteConfirmPending bool
	pendingContent          string
	baselineContent         string
	statusMsg               string
	statusExpiry            time.Time
}

func NewSummaryModel(content, project, meetingID string, store *storage.Store, width, height int) SummaryModel {
	input := textarea.New()
	input.SetWidth(width)
	input.SetHeight(summaryTextareaHeight(height))
	input.Blur()

	if content != "" {
		input.SetValue(content)
	}

	return SummaryModel{
		textarea:                input,
		store:                   store,
		project:                 project,
		meetingID:               meetingID,
		width:                   width,
		height:                  height,
		dirty:                   false,
		overwriteConfirmPending: false,
		pendingContent:          "",
		baselineContent:         input.Value(),
	}
}

func (m SummaryModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m SummaryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.statusMsg != "" && time.Now().After(m.statusExpiry) {
		m.statusMsg = ""
	}

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.textarea.SetWidth(m.width)
		m.textarea.SetHeight(summaryTextareaHeight(m.height))
		return m, nil
	case SummarySaveDoneMsg:
		now := time.Now()
		m.statusMsg = "Saved"
		m.statusExpiry = now.Add(3 * time.Second)
		m.dirty = false
		m.baselineContent = m.textarea.Value()
		return m, nil
	case SummarySaveErrMsg:
		now := time.Now()
		m.statusMsg = "Save error: " + typedMsg.Err.Error()
		m.statusExpiry = now.Add(5 * time.Second)
		return m, nil
	case tea.KeyPressMsg:
		if m.overwriteConfirmPending {
			switch typedMsg.String() {
			case "y":
				m.AcceptOverwrite()
			case "n":
				m.RejectOverwrite()
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.RejectOverwrite()
				return m, nil
			}

			return m, nil
		}

		if typedMsg.String() == "ctrl+s" {
			return m, m.saveCmd()
		}

		updatedTextarea, cmd := m.textarea.Update(msg)
		m.textarea = updatedTextarea
		m.dirty = m.textarea.Value() != m.baselineContent
		return m, cmd
	default:
		updatedTextarea, cmd := m.textarea.Update(msg)
		m.textarea = updatedTextarea
		m.dirty = m.textarea.Value() != m.baselineContent
		return m, cmd
	}
}

func (m SummaryModel) View() tea.View {
	headerText := "── AI Summary"
	if m.dirty {
		headerText += " •"
	}
	if m.overwriteConfirmPending {
		headerText += " [New AI content available: y=accept, n=keep yours]"
	}
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		headerText += " [" + m.statusMsg + "]"
	}

	header := lipgloss.NewStyle().Faint(true).Render(headerText)
	return tea.NewView(header + "\n" + m.textarea.View())
}

func (m *SummaryModel) SetContent(md string) {
	if !m.dirty {
		m.textarea.SetValue(md)
		m.baselineContent = md
		m.dirty = false
		m.overwriteConfirmPending = false
		m.pendingContent = ""
		return
	}

	m.overwriteConfirmPending = true
	m.pendingContent = md
}

func (m SummaryModel) Value() string {
	return m.textarea.Value()
}

func (m SummaryModel) IsDirty() bool {
	return m.dirty
}

func (m SummaryModel) IsOverwritePending() bool {
	return m.overwriteConfirmPending
}

func (m *SummaryModel) AcceptOverwrite() {
	m.textarea.SetValue(m.pendingContent)
	m.baselineContent = m.pendingContent
	m.dirty = false
	m.overwriteConfirmPending = false
	m.pendingContent = ""
}

func (m *SummaryModel) RejectOverwrite() {
	m.overwriteConfirmPending = false
	m.pendingContent = ""
}

func (m *SummaryModel) Focus() {
	m.textarea.Focus()
}

func (m *SummaryModel) Blur() {
	m.textarea.Blur()
}

func (m SummaryModel) saveCmd() tea.Cmd {
	content := m.textarea.Value()

	return func() tea.Msg {
		if m.store == nil {
			return SummarySaveErrMsg{Err: errors.New("summary store is not configured")}
		}
		if m.project == "" {
			return SummarySaveErrMsg{Err: errors.New("summary project is empty")}
		}
		if m.meetingID == "" {
			return SummarySaveErrMsg{Err: errors.New("summary meeting ID is empty")}
		}

		err := m.store.SaveStructuredNotes(m.project, m.meetingID, content)
		if err != nil {
			return SummarySaveErrMsg{Err: err}
		}

		return SummarySaveDoneMsg{}
	}
}

func summaryTextareaHeight(totalHeight int) int {
	textareaHeight := totalHeight - 2
	if textareaHeight < 1 {
		return 1
	}

	return textareaHeight
}
