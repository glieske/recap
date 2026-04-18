package ai

import (
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestPromptsStructurePromptMessagesAndContent(t *testing.T) {
	meta := MeetingMeta{
		Title:        "Sprint Planning",
		Date:         "2026-04-16",
		Participants: []string{"Alice", "Bob"},
		TicketID:     "ABC-123",
	}
	rawNotes := "Discussed backlog priorities and deployment timing."

	messages := StructurePrompt(rawNotes, meta)

	if got, want := len(messages), 2; got != want {
		t.Fatalf("StructurePrompt message count = %d, want %d", got, want)
	}

	if got, want := messages[0].Role, openai.ChatMessageRoleSystem; got != want {
		t.Fatalf("StructurePrompt system role = %q, want %q", got, want)
	}

	if got, want := messages[1].Role, openai.ChatMessageRoleUser; got != want {
		t.Fatalf("StructurePrompt user role = %q, want %q", got, want)
	}

	system := messages[0].Content
	requiredHeadings := []string{
		"## Summary",
		"## Attendees",
		"## Key Decisions",
		"## Action Points",
		"## Discussion Notes",
		"## Next Steps",
	}
	for _, heading := range requiredHeadings {
		if !strings.Contains(system, heading) {
			t.Fatalf("system prompt missing required heading %q", heading)
		}
	}

	user := messages[1].Content
	requiredUserParts := []string{
		"Title: Sprint Planning",
		"Date: 2026-04-16",
		"Participants: Alice, Bob",
		"Ticket ID: ABC-123",
		"Raw Notes:\nDiscussed backlog priorities and deployment timing.",
	}
	for _, part := range requiredUserParts {
		if !strings.Contains(user, part) {
			t.Fatalf("user prompt missing required content %q", part)
		}
	}
}

func TestPromptsStructurePromptTicketIDAndParticipantsEdgeCases(t *testing.T) {
	t.Run("empty TicketID omits line", func(t *testing.T) {
		meta := MeetingMeta{
			Title:        "Retrospective",
			Date:         "2026-04-17",
			Participants: []string{"Eve"},
			TicketID:     "",
		}

		messages := StructurePrompt("raw", meta)
		if got, want := len(messages), 2; got != want {
			t.Fatalf("StructurePrompt message count = %d, want %d", got, want)
		}

		user := messages[1].Content
		if strings.Contains(user, "Ticket ID:") {
			t.Fatalf("user prompt unexpectedly contains Ticket ID line: %q", user)
		}
		if !strings.Contains(user, "Participants: Eve") {
			t.Fatalf("user prompt missing participants line, got %q", user)
		}
	})

	t.Run("nil participants yields empty participants line", func(t *testing.T) {
		meta := MeetingMeta{Title: "Nil Participants", Date: "2026-04-18", Participants: nil}
		user := StructurePrompt("notes", meta)[1].Content

		if !strings.Contains(user, "Participants: ") {
			t.Fatalf("expected participants line in user prompt, got %q", user)
		}
		if !strings.Contains(user, "Participants: \n\nRaw Notes:\nnotes") {
			t.Fatalf("expected empty participants rendering, got %q", user)
		}
	})

	t.Run("empty participants slice yields empty participants line", func(t *testing.T) {
		meta := MeetingMeta{Title: "Empty Participants", Date: "2026-04-18", Participants: []string{}}
		user := StructurePrompt("notes", meta)[1].Content

		if !strings.Contains(user, "Participants: ") {
			t.Fatalf("expected participants line in user prompt, got %q", user)
		}
		if !strings.Contains(user, "Participants: \n\nRaw Notes:\nnotes") {
			t.Fatalf("expected empty participants rendering, got %q", user)
		}
	})
}

func TestPromptsEmailPromptMessagesAndVerbatimInput(t *testing.T) {
	structuredMD := "## Summary\nDone.\n\n## Next Steps\n- Ship"

	messages := EmailPrompt(structuredMD, "en")

	if got, want := len(messages), 2; got != want {
		t.Fatalf("EmailPrompt message count = %d, want %d", got, want)
	}

	if got, want := messages[0].Role, openai.ChatMessageRoleSystem; got != want {
		t.Fatalf("EmailPrompt system role = %q, want %q", got, want)
	}

	if got, want := messages[1].Role, openai.ChatMessageRoleUser; got != want {
		t.Fatalf("EmailPrompt user role = %q, want %q", got, want)
	}

	if !strings.Contains(messages[0].Content, "summary paragraph") {
		t.Fatalf("EmailPrompt system prompt must include summary paragraph requirement, got %q", messages[0].Content)
	}

	if strings.Contains(messages[0].Content, "Subject:") {
		t.Fatalf("EmailPrompt system prompt must not include Subject requirement, got %q", messages[0].Content)
	}

	if got, want := messages[1].Content, structuredMD; got != want {
		t.Fatalf("EmailPrompt user content mismatch\ngot:  %q\nwant: %q", got, want)
	}

	if !strings.Contains(messages[0].Content, "English") {
		t.Fatalf("EmailPrompt system prompt must include language instruction, got %q", messages[0].Content)
	}
}

func TestPromptsConstants(t *testing.T) {
	if got, want := StructureTemperature, float32(0.2); got != want {
		t.Fatalf("StructureTemperature = %v, want %v", got, want)
	}

	if got, want := EmailTemperature, float32(0.4); got != want {
		t.Fatalf("EmailTemperature = %v, want %v", got, want)
	}
}

func TestPromptsBuildStructurePromptUserMessageWhitespaceTicketIDAndBoundaryInput(t *testing.T) {
	meta := MeetingMeta{
		Title:        "Quarterly Review 🚀",
		Date:         "2026-04-19",
		Participants: []string{"Ana", "李雷"},
		TicketID:     "   \t\n",
	}
	rawNotes := "Line 1 with unicode ✅\nLine 2 with special chars <> ${x} ../"

	message := buildStructurePromptUserMessage(rawNotes, meta)

	if strings.Contains(message, "Ticket ID:") {
		t.Fatalf("whitespace-only TicketID should be treated as empty, got %q", message)
	}

	expectedParts := []string{
		"Title: Quarterly Review 🚀",
		"Date: 2026-04-19",
		"Participants: Ana, 李雷",
		rawNotes,
	}
	for _, part := range expectedParts {
		if !strings.Contains(message, part) {
			t.Fatalf("helper output missing expected content %q", part)
		}
	}
}
