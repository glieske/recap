package tui

import (
	"fmt"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const previewPlaceholder = "No structured notes yet. Press Ctrl+A in the editor to generate."

type PreviewModel struct {
	viewport viewport.Model
	content  string
	title    string
	ready    bool
	width    int
	height   int
}

func NewPreviewModel(content string, title string, width, height int) PreviewModel {
	normalizedContent := content
	if normalizedContent == "" {
		normalizedContent = previewPlaceholder
	}

	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(previewViewportHeight(height)))
	vp.SetContent(normalizedContent)

	return PreviewModel{
		viewport: vp,
		content:  normalizedContent,
		title:    title,
		ready:    true,
		width:    width,
		height:   height,
	}
}

func (m PreviewModel) Init() tea.Cmd {
	return nil
}

func (m PreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.viewport.SetWidth(m.width)
		m.viewport.SetHeight(previewViewportHeight(m.height))
		m.ready = true
		return m, nil
	case tea.KeyPressMsg:
		switch typedMsg.String() {
		case "esc", "ctrl+p", "q":
			return m, func() tea.Msg { return TogglePreviewMsg{} }
		default:
			updatedViewport, cmd := m.viewport.Update(msg)
			m.viewport = updatedViewport
			return m, cmd
		}
	default:
		updatedViewport, cmd := m.viewport.Update(msg)
		m.viewport = updatedViewport
		return m, cmd
	}
}

func (m PreviewModel) View() tea.View {
	headerStyle := lipgloss.NewStyle().Faint(true)
	footerStyle := lipgloss.NewStyle().Faint(true)

	header := headerStyle.Render(fmt.Sprintf("── Preview: %s ──", m.title))
	footer := footerStyle.Render("↑↓ scroll • Esc back")

	return tea.NewView(header + "\n" + m.viewport.View() + "\n" + footer)
}

func (m *PreviewModel) SetContent(content string) {
	if content == "" {
		content = previewPlaceholder
	}

	m.content = content
	m.viewport.SetContent(content)
}

func previewViewportHeight(totalHeight int) int {
	viewportHeight := totalHeight - 2
	if viewportHeight < 1 {
		return 1
	}

	return viewportHeight
}
