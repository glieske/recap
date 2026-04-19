package ai

import (
	"strings"
	"testing"
)

func TestPromptsEmailPromptLanguageSelection(t *testing.T) {
	tests := []struct {
		name             string
		language         string
		wantLanguageName string
	}{
		{name: "en uses English", language: "en", wantLanguageName: "English"},
		{name: "pl uses Polish", language: "pl", wantLanguageName: "Polish"},
		{name: "no uses Norwegian", language: "no", wantLanguageName: "Norwegian"},
		{name: "empty defaults to English", language: "", wantLanguageName: "English"},
		{name: "invalid defaults to English", language: "xyz", wantLanguageName: "English"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := EmailPrompt("notes", tt.language)

			if got, want := len(messages), 2; got != want {
				t.Fatalf("EmailPrompt message count = %d, want %d", got, want)
			}

			system := messages[0].Content
			if !strings.Contains(system, tt.wantLanguageName) {
				t.Fatalf("EmailPrompt system message = %q, want to contain %q", system, tt.wantLanguageName)
			}
		})
	}
}

func TestPromptsEmailLanguageNameCaseAndWhitespaceInvariant(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "en", want: "English"},
		{input: " EN ", want: "English"},
		{input: "pl", want: "Polish"},
		{input: "\tpl\n", want: "Polish"},
		{input: "NO", want: "Norwegian"},
		{input: "  no  ", want: "Norwegian"},
		{input: "fr", want: "English"},
	}

	for _, tt := range tests {
		if got := emailLanguageName(tt.input); got != tt.want {
			t.Fatalf("emailLanguageName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
