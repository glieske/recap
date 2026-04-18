package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/glieske/recap/internal/storage"
)

// AutoSaveTickMsg signals periodic auto-save checks.
type AutoSaveTickMsg struct{}

// SaveDoneMsg signals successful save completion.
type SaveDoneMsg struct{}

// SaveErrMsg signals save failure.
type SaveErrMsg struct {
	Err error
}

// TriggerAIMsg requests AI structuring from parent model.
type TriggerAIMsg struct{}

// TriggerEmailMsg requests email generation from parent model.
type TriggerEmailMsg struct{}

// TogglePreviewMsg requests preview pane toggle from parent model.
type TogglePreviewMsg struct{}

type EditorModel struct {
	textarea        textarea.Model
	splitMode       bool
	focusedPane     int
	summaryModel    SummaryModel
	hasSummaryModel bool
	meeting         *storage.Meeting
	store           *storage.Store
	providerName    string
	providerModel   string
	width           int
	height          int
	dirty           bool
	lastSaved       time.Time
	startTime       time.Time
	statusMsg       string
	statusExpiry    time.Time
	err             error
}

func NewEditorModel(meeting *storage.Meeting, store *storage.Store, width, height int, providerName, providerModel string) EditorModel {
	now := time.Now()
	input := textarea.New()
	input.SetWidth(width)
	input.SetHeight(maxEditorHeight(height))
	input.Focus()

	if store != nil && meeting != nil {
		rawNotes, err := store.LoadRawNotes(meeting.Project, meeting.ID)
		if err == nil {
			input.SetValue(rawNotes)
		}
	}

	return EditorModel{
		textarea:      input,
		meeting:       meeting,
		store:         store,
		providerName:  providerName,
		providerModel: providerModel,
		width:         width,
		height:        height,
		dirty:         false,
		lastSaved:     now,
		startTime:     now,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, autoSaveTick())
}

func (m *EditorModel) SetSummaryModel(sm SummaryModel) {
	m.summaryModel = sm
	m.hasSummaryModel = true
}

func (m EditorModel) GetSummaryModel() SummaryModel {
	return m.summaryModel
}

func (m EditorModel) IsSplitMode() bool {
	return m.splitMode
}

func (m *EditorModel) ActivateSplitMode() {
	if m.splitMode || !m.hasSummaryModel || m.width < 100 {
		return
	}

	m.splitMode = true
	innerPaneHeight := maxEditorHeight(m.height) - 2
	if innerPaneHeight < 1 {
		innerPaneHeight = 1
	}
	m.focusedPane = 1
	m.textarea.Blur()
	m.summaryModel.Focus()
	m.textarea.SetHeight(innerPaneHeight)
	leftWidth := (m.width - 1) / 2
	rightWidth := m.width - leftWidth - 1
	leftContentWidth := leftWidth - 2
	rightContentWidth := rightWidth - 2
	if leftContentWidth < 1 {
		leftContentWidth = 1
	}
	if rightContentWidth < 1 {
		rightContentWidth = 1
	}
	m.textarea.SetWidth(leftContentWidth)
	updatedSummaryModel, _ := m.summaryModel.Update(tea.WindowSizeMsg{Width: rightContentWidth, Height: innerPaneHeight})
	typedSummaryModel, ok := updatedSummaryModel.(SummaryModel)
	if ok {
		m.summaryModel = typedSummaryModel
	}
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.statusMsg != "" && time.Now().After(m.statusExpiry) {
		m.statusMsg = ""
	}

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		fullEditorHeight := maxEditorHeight(m.height)
		if m.splitMode && m.hasSummaryModel {
			innerPaneHeight := fullEditorHeight - 2
			if innerPaneHeight < 1 {
				innerPaneHeight = 1
			}
			m.textarea.SetHeight(innerPaneHeight)
			leftWidth := (m.width - 1) / 2
			rightWidth := m.width - leftWidth - 1
			leftContentWidth := leftWidth - 2
			rightContentWidth := rightWidth - 2
			if leftContentWidth < 1 {
				leftContentWidth = 1
			}
			if rightContentWidth < 1 {
				rightContentWidth = 1
			}
			m.textarea.SetWidth(leftContentWidth)
			updatedSummaryModel, _ := m.summaryModel.Update(tea.WindowSizeMsg{Width: rightContentWidth, Height: innerPaneHeight})
			typedSummaryModel, ok := updatedSummaryModel.(SummaryModel)
			if ok {
				m.summaryModel = typedSummaryModel
			}
		} else {
			m.textarea.SetHeight(fullEditorHeight)
			m.textarea.SetWidth(m.width)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.splitMode && m.hasSummaryModel && m.focusedPane == 1 {
			key := typedMsg.String()
			if key != "tab" && key != "ctrl+p" && key != "ctrl+s" && key != "esc" && key != "ctrl+a" && key != "ctrl+e" && key != "ctrl+t" {
				updatedSummaryModel, cmd := m.summaryModel.Update(msg)
				typedSummaryModel, ok := updatedSummaryModel.(SummaryModel)
				if ok {
					m.summaryModel = typedSummaryModel
				}
				return m, cmd
			}
		}

		switch typedMsg.String() {
		case "ctrl+t":
			m.textarea.InsertString(fmt.Sprintf("[%s] ", time.Now().Format("15:04")))
			m.dirty = true
			return m, nil
		case "ctrl+s":
			m.dirty = false
			return m, m.saveCmd()
		case "ctrl+a":
			return m, func() tea.Msg { return TriggerAIMsg{} }
		case "ctrl+e":
			return m, func() tea.Msg { return TriggerEmailMsg{} }
		case "ctrl+p":
			if !m.hasSummaryModel {
				return m, func() tea.Msg { return TogglePreviewMsg{} }
			}
			if m.width < 100 {
				now := time.Now()
				m.statusMsg = "Terminal too narrow for split view (need 100+ cols)"
				m.statusExpiry = now.Add(3 * time.Second)
				return m, nil
			}

			m.splitMode = !m.splitMode
			if m.splitMode {
				innerPaneHeight := maxEditorHeight(m.height) - 2
				if innerPaneHeight < 1 {
					innerPaneHeight = 1
				}
				m.focusedPane = 0
				m.textarea.Focus()
				m.summaryModel.Blur()
				m.textarea.SetHeight(innerPaneHeight)
				leftWidth := (m.width - 1) / 2
				rightWidth := m.width - leftWidth - 1
				leftContentWidth := leftWidth - 2
				rightContentWidth := rightWidth - 2
				if leftContentWidth < 1 {
					leftContentWidth = 1
				}
				if rightContentWidth < 1 {
					rightContentWidth = 1
				}
				m.textarea.SetWidth(leftContentWidth)
				updatedSummaryModel, _ := m.summaryModel.Update(tea.WindowSizeMsg{Width: rightContentWidth, Height: innerPaneHeight})
				typedSummaryModel, ok := updatedSummaryModel.(SummaryModel)
				if ok {
					m.summaryModel = typedSummaryModel
				}
			} else {
				m.focusedPane = 0
				m.textarea.SetHeight(maxEditorHeight(m.height))
				m.textarea.SetWidth(m.width)
				m.textarea.Focus()
				m.summaryModel.Blur()
			}

			return m, nil
		case "tab":
			if m.splitMode && m.hasSummaryModel {
				if m.focusedPane == 0 {
					m.textarea.Blur()
					m.summaryModel.Focus()
					m.focusedPane = 1
				} else {
					m.summaryModel.Blur()
					m.textarea.Focus()
					m.focusedPane = 0
				}
				return m, nil
			}
			updatedTextarea, cmd := m.textarea.Update(msg)
			m.textarea = updatedTextarea
			m.dirty = true
			return m, cmd
		case "esc":
			if m.splitMode {
				m.splitMode = false
				m.focusedPane = 0
				m.textarea.Focus()
				m.summaryModel.Blur()
				m.textarea.SetHeight(maxEditorHeight(m.height))
				m.textarea.SetWidth(m.width)
				return m, nil
			}
			updatedTextarea, cmd := m.textarea.Update(msg)
			m.textarea = updatedTextarea
			return m, cmd
		default:
			updatedTextarea, cmd := m.textarea.Update(msg)
			m.textarea = updatedTextarea
			m.dirty = true
			return m, cmd
		}
	case AutoSaveTickMsg:
		if m.dirty && m.store != nil && m.meeting != nil {
			m.dirty = false
			return m, tea.Batch(m.saveCmd(), autoSaveTick())
		}

		return m, autoSaveTick()
	case SaveDoneMsg:
		now := time.Now()
		m.lastSaved = now
		m.statusMsg = "Saved"
		m.statusExpiry = now.Add(3 * time.Second)
		return m, nil
	case SaveErrMsg:
		now := time.Now()
		m.dirty = true
		m.err = typedMsg.Err
		m.statusMsg = "Save error: " + typedMsg.Err.Error()
		m.statusExpiry = now.Add(5 * time.Second)
		return m, nil
	}

	updatedTextarea, cmd := m.textarea.Update(msg)
	m.textarea = updatedTextarea
	return m, cmd
}

func (m EditorModel) View() tea.View {
	statusBar := m.renderStatusBar()
	legend := m.renderLegend()
	if m.splitMode && m.hasSummaryModel {
		leftWidth := (m.width - 1) / 2
		rightWidth := m.width - leftWidth - 1
		leftContentWidth := leftWidth - 2
		rightContentWidth := rightWidth - 2
		if leftContentWidth < 1 {
			leftContentWidth = 1
		}
		if rightContentWidth < 1 {
			rightContentWidth = 1
		}

		focusedStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("212"))
		unfocusedStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("241"))
		leftPaneStyle := unfocusedStyle
		rightPaneStyle := unfocusedStyle
		if m.focusedPane == 0 {
			leftPaneStyle = focusedStyle
		} else {
			rightPaneStyle = focusedStyle
		}

		leftView := m.textarea.View()
		rightView := safeViewContent(func() tea.View { return m.summaryModel.View() })
		leftPane := leftPaneStyle.Width(leftContentWidth).Render(leftView)
		rightPane := rightPaneStyle.Width(rightContentWidth).Render(rightView)
		dividerView := renderVerticalDivider(maxEditorHeight(m.height))
		splitContent := lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPane,
			dividerView,
			rightPane,
		)
		return tea.NewView(splitContent + "\n" + statusBar + "\n" + legend)
	}

	return tea.NewView(m.textarea.View() + "\n" + statusBar + "\n" + legend)
}

func (m EditorModel) renderLegend() string {
	legendText := "ctrl+s Save | ctrl+a AI | ctrl+p Preview | ctrl+t Timestamp | ctrl+e Email | ctrl+o Provider | ctrl+/ Help"
	if m.splitMode && m.focusedPane == 0 {
		legendText = "Tab Switch | ctrl+s Save | ctrl+a AI | ctrl+t Timestamp | Esc Collapse | ctrl+/ Help"
	} else if m.splitMode && m.focusedPane == 1 {
		legendText = "Tab Switch | ctrl+s Save Summary | ctrl+e Email | Esc Collapse | ctrl+/ Help"
	}
	barWidth := m.width
	if barWidth < 1 {
		barWidth = 1
	}

	legendRunes := []rune(legendText)
	if len(legendRunes) > barWidth {
		if barWidth == 1 {
			legendText = "…"
		} else {
			legendText = string(legendRunes[:barWidth-1]) + "…"
		}
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(barWidth).
		Render(legendText)
}

func (m EditorModel) saveCmd() tea.Cmd {
	content := m.textarea.Value()

	return func() tea.Msg {
		if m.store == nil || m.meeting == nil {
			return SaveErrMsg{Err: errors.New("editor not configured")}
		}

		err := m.store.SaveRawNotes(m.meeting.Project, m.meeting.ID, content)
		if err != nil {
			return SaveErrMsg{Err: err}
		}

		return SaveDoneMsg{}
	}
}

func autoSaveTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return AutoSaveTickMsg{}
	})
}

func (m EditorModel) renderStatusBar() string {
	left := "- | -"
	if m.meeting != nil {
		left = fmt.Sprintf("%s | %s", m.meeting.TicketID, m.meeting.Title)
	}

	elapsed := time.Since(m.startTime).Truncate(time.Second)
	center := formatElapsed(elapsed)

	rightPrefix := ""
	if m.providerName != "" && m.providerModel != "" {
		rightPrefix = m.providerName + " / " + m.providerModel
	} else if m.providerName != "" {
		rightPrefix = m.providerName
	}

	right := fmt.Sprintf("%d chars", len(m.textarea.Value()))
	if rightPrefix != "" {
		right = rightPrefix + " | " + right
	}
	if m.dirty {
		right += " •"
	}
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		right += " | " + m.statusMsg
	}

	barWidth := m.width
	if barWidth < 1 {
		barWidth = 1
	}

	content := composeStatusLine(left, center, right, barWidth)

	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Width(barWidth).
		Render(content)
}

func composeStatusLine(left, center, right string, width int) string {
	if width <= 0 {
		return ""
	}

	leftLen := len([]rune(left))
	centerLen := len([]rune(center))
	rightLen := len([]rune(right))
	total := leftLen + centerLen + rightLen
	if total <= width {
		remaining := width - total
		leftGap := remaining / 2
		rightGap := remaining - leftGap
		line := left + fmt.Sprintf("%*s", leftGap, "") + center + fmt.Sprintf("%*s", rightGap, "") + right
		lineLen := len([]rune(line))
		if lineLen < width {
			return fmt.Sprintf("%-*s", width, line)
		}

		if lineLen > width {
			return string([]rune(line)[:width])
		}

		return line
	}

	line := left + " | " + center + " | " + right
	lineLen := len([]rune(line))
	if lineLen > width {
		return string([]rune(line)[:width])
	}

	return fmt.Sprintf("%-*s", width, line)
}

func formatElapsed(elapsed time.Duration) string {
	totalSeconds := int(elapsed.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func maxEditorHeight(totalHeight int) int {
	textareaHeight := totalHeight - 4
	if textareaHeight < 1 {
		return 1
	}

	return textareaHeight
}

func renderVerticalDivider(height int) string {
	if height < 1 {
		return ""
	}
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	return strings.Join(lines, "\n")
}
