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
		{name: "de uses German", language: "de", wantLanguageName: "German"},
		{name: "zh uses Chinese", language: "zh", wantLanguageName: "Chinese"},
		{name: "es uses Spanish", language: "es", wantLanguageName: "Spanish"},
		{name: "fr uses French", language: "fr", wantLanguageName: "French"},
		{name: "ar uses Arabic", language: "ar", wantLanguageName: "Arabic"},
		{name: "bn uses Bengali", language: "bn", wantLanguageName: "Bengali"},
		{name: "pt uses Portuguese", language: "pt", wantLanguageName: "Portuguese"},
		{name: "ru uses Russian", language: "ru", wantLanguageName: "Russian"},
		{name: "ja uses Japanese", language: "ja", wantLanguageName: "Japanese"},
		{name: "hi uses Hindi", language: "hi", wantLanguageName: "Hindi"},
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

func TestPromptsEmailPromptCaseAndWhitespaceInvariant(t *testing.T) {
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
		{input: "DE", want: "German"},
		{input: " de ", want: "German"},
		{input: "unknown", want: "English"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			messages := EmailPrompt("notes", tt.input)
			system := messages[0].Content
			if !strings.Contains(system, tt.want) {
				t.Fatalf("EmailPrompt(%q) system message does not contain %q", tt.input, tt.want)
			}
		})
	}
}
