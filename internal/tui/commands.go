package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
)

// AIStructureDoneMsg signals AI structuring completed successfully.
type AIStructureDoneMsg struct {
	StructuredMD string
}

// AIStructureErrMsg signals AI structuring failed.
type AIStructureErrMsg struct {
	Err error
}

// AIEmailDoneMsg signals email generation completed successfully.
type AIEmailDoneMsg struct {
	Subject string
	Body    string
}

// AIEmailErrMsg signals email generation failed.
type AIEmailErrMsg struct {
	Err error
}

// AIProgressMsg signals AI operation is in progress (for spinner/status).
type AIProgressMsg struct {
	Operation string // "structuring" or "generating email"
}

// StructureNotesCmd runs note structuring via configured AI provider.
func StructureNotesCmd(provider ai.Provider, rawNotes string, meta ai.MeetingMeta) tea.Cmd {
	if provider == nil {
		return func() tea.Msg {
			return AIStructureErrMsg{Err: errors.New("no AI provider configured")}
		}
	}

	if strings.TrimSpace(rawNotes) == "" {
		return func() tea.Msg {
			return AIStructureErrMsg{Err: errors.New("no raw notes to structure")}
		}
	}

	return func() tea.Msg {
		structuredMD, err := provider.StructureNotes(context.Background(), rawNotes, meta)
		if err != nil {
			return AIStructureErrMsg{Err: err}
		}

		return AIStructureDoneMsg{StructuredMD: structuredMD}
	}
}

// GenerateEmailCmd runs email summary generation via configured AI provider.
func GenerateEmailCmd(provider ai.Provider, structuredMD string, language string) tea.Cmd {
	if provider == nil {
		return func() tea.Msg {
			return AIEmailErrMsg{Err: errors.New("no AI provider configured")}
		}
	}

	if strings.TrimSpace(structuredMD) == "" {
		return func() tea.Msg {
			return AIEmailErrMsg{Err: errors.New("no structured notes available")}
		}
	}

	return func() tea.Msg {
		emailResponse, err := provider.GenerateEmailSummary(context.Background(), structuredMD, language)
		if err != nil {
			return AIEmailErrMsg{Err: err}
		}

		subject, body := parseEmailResponse(emailResponse)
		return AIEmailDoneMsg{
			Subject: subject,
			Body:    body,
		}
	}
}

func parseEmailResponse(response string) (subject, body string) {
	trimmedResponse := strings.TrimSpace(response)
	lines := strings.Split(trimmedResponse, "\n")
	if len(lines) == 0 {
		return "Meeting Summary", ""
	}

	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "Subject: ") {
		return "Meeting Summary", trimmedResponse
	}

	subject = strings.TrimSpace(strings.TrimPrefix(firstLine, "Subject: "))
	bodyStart := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "" {
			bodyStart = idx + 1
			break
		}
	}

	if bodyStart < 0 {
		bodyStart = 1
	}

	if bodyStart >= len(lines) {
		return strings.TrimSpace(subject), ""
	}

	body = strings.TrimSpace(strings.Join(lines[bodyStart:], "\n"))
	return strings.TrimSpace(subject), body
}
