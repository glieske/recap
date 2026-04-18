package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type DismissModalMsg struct{}

func RenderOverlay(background string, modal string, width, height int) string {
	dimStyle := lipgloss.NewStyle().Faint(true)
	bgLines := strings.Split(background, "\n")
	for i, line := range bgLines {
		bgLines[i] = dimStyle.Render(line)
	}

	if width <= 0 || height <= 0 {
		return modal
	}

	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}
	if len(bgLines) > height {
		bgLines = bgLines[:height]
	}

	modalLines := strings.Split(modal, "\n")
	modalWidth := 0
	for _, line := range modalLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > modalWidth {
			modalWidth = lineWidth
		}
	}

	startX := (width - modalWidth) / 2
	startY := (height - len(modalLines)) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, modalLine := range modalLines {
		targetY := startY + i
		if targetY < 0 || targetY >= len(bgLines) {
			continue
		}

		bgLine := bgLines[targetY]
		modalLineWidth := lipgloss.Width(modalLine)

		leftPart := ansi.Truncate(bgLine, startX, "")
		leftWidth := lipgloss.Width(leftPart)
		if leftWidth < startX {
			leftPart += strings.Repeat(" ", startX-leftWidth)
		}

		rightStart := startX + modalLineWidth
		rightPart := ""
		if rightStart < width {
			rightPart = ansi.TruncateLeft(bgLine, rightStart, "")
		}

		bgLines[targetY] = leftPart + modalLine + rightPart
	}

	return strings.Join(bgLines, "\n")
}

type ModalModel struct {
	Content tea.Model
	Title   string
	width   int
	height  int
}

func NewModalModel(title string, content tea.Model, width, height int) ModalModel {
	return ModalModel{
		Content: content,
		Title:   title,
		width:   width,
		height:  height,
	}
}

func (m ModalModel) Init() tea.Cmd {
	if m.Content == nil {
		return nil
	}

	return m.Content.Init()
}

func (m ModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		if m.Content == nil {
			return m, nil
		}

		updatedContent, command := m.Content.Update(msg)
		m.Content = updatedContent
		return m, command
	case tea.KeyPressMsg:
		if typedMsg.String() == "esc" {
			return m, func() tea.Msg { return DismissModalMsg{} }
		}

		if m.Content == nil {
			return m, nil
		}

		updatedContent, command := m.Content.Update(msg)
		m.Content = updatedContent
		return m, command
	default:
		if m.Content == nil {
			return m, nil
		}

		updatedContent, command := m.Content.Update(msg)
		m.Content = updatedContent
		return m, command
	}
}

func (m ModalModel) View() tea.View {
	contentView := ""
	if m.Content != nil {
		contentView = m.Content.View().Content
	}

	maxBoxWidth := (m.width * 60) / 100
	maxBoxHeight := (m.height * 70) / 100
	if maxBoxWidth < 1 {
		maxBoxWidth = 1
	}
	if maxBoxHeight < 1 {
		maxBoxHeight = 1
	}

	innerMaxWidth := maxBoxWidth - 4
	if innerMaxWidth < 1 {
		innerMaxWidth = 1
	}

	contentLines := strings.Split(contentView, "\n")
	titleWidth := lipgloss.Width(m.Title)
	contentWidth := 0
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > contentWidth {
			contentWidth = lineWidth
		}
	}

	innerWidth := titleWidth
	if contentWidth > innerWidth {
		innerWidth = contentWidth
	}
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerWidth > innerMaxWidth {
		innerWidth = innerMaxWidth
	}

	innerHeight := maxBoxHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	titleLine := lipgloss.NewStyle().Bold(true).Width(innerWidth).Align(lipgloss.Center).Render(m.Title)
	separatorLine := strings.Repeat("─", innerWidth)

	renderedContentLines := make([]string, 0, len(contentLines))
	for _, line := range contentLines {
		renderedContentLines = append(renderedContentLines, ansi.Truncate(line, innerWidth, ""))
	}

	bodyLines := make([]string, 0, len(renderedContentLines)+2)
	bodyLines = append(bodyLines, titleLine, separatorLine)
	bodyLines = append(bodyLines, renderedContentLines...)
	if len(bodyLines) > innerHeight {
		bodyLines = bodyLines[:innerHeight]
	}

	body := strings.Join(bodyLines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(innerWidth + 4).
		Render(body)

	return tea.NewView(box)
}
