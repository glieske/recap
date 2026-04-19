package ai

import (
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	StructureTemperature float32 = 0.2
	EmailTemperature     float32 = 0.4
)

// StructurePrompt builds chat messages for converting raw notes into structured Markdown.
func StructurePrompt(rawNotes string, meta MeetingMeta) []openai.ChatCompletionMessage {
	systemPrompt := strings.TrimSpace(`You are a meeting notes assistant.

Transform the provided raw meeting notes into clear, structured Markdown using EXACTLY these sections and headings:
## Summary
## Attendees
## Key Decisions
## Action Points
## Discussion Notes
## Next Steps

Formatting requirements:
- ## Summary: 2-3 sentence overview.
- ## Attendees: bullet list derived from participants.
- ## Key Decisions: numbered list.
- ## Action Points: Markdown table with exactly these columns: Task | Due Date.
- ## Discussion Notes: detailed notes organized by topic.
- ## Next Steps: bullet list.

Output requirements:
- Output ONLY valid Markdown.
- Do NOT use code fences.
- Do NOT include preamble or commentary outside the requested sections.
- Do NOT assign or attribute tasks to specific participants unless the raw notes explicitly state who is responsible.`)

	userPrompt := buildStructurePromptUserMessage(rawNotes, meta)

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}
}

// EmailPrompt builds chat messages for converting structured notes into email body content ready to paste.
func EmailPrompt(structuredMD string, language string) []openai.ChatCompletionMessage {
	systemPrompt := strings.TrimSpace(fmt.Sprintf(`You are an executive communications assistant.

Generate plain-text email body content from the provided structured meeting notes.

Required email structure:
1) Brief summary paragraph
2) Action points section as a bullet list (NOT a table)
3) Next steps

Output requirements:
- Output plain text email only (not Markdown).
- The content must be ready to paste into an email body as-is.
- Do not include commentary about the format.

Language: Write the entire email body in %s.`, emailLanguageName(language)))

	return []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: structuredMD,
		},
	}
}

func emailLanguageName(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "en":
		return "English"
	case "pl":
		return "Polish"
	case "no":
		return "Norwegian"
	default:
		return "English"
	}
}

func buildStructurePromptUserMessage(rawNotes string, meta MeetingMeta) string {
	participantsLine := strings.Join(meta.Participants, ", ")

	if strings.TrimSpace(meta.TicketID) == "" {
		return fmt.Sprintf(
			"Title: %s\nDate: %s\nParticipants: %s\n\nRaw Notes:\n%s",
			meta.Title,
			meta.Date,
			participantsLine,
			rawNotes,
		)
	}

	return fmt.Sprintf(
		"Title: %s\nDate: %s\nTicket ID: %s\nParticipants: %s\n\nRaw Notes:\n%s",
		meta.Title,
		meta.Date,
		meta.TicketID,
		participantsLine,
		rawNotes,
	)
}
