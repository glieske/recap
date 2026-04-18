package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/glieske/recap/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

func TestAdversarialConfigInjection_OpenRouterAPIKey(t *testing.T) {
	t.Parallel()

	veryLong := strings.Repeat("k", 12*1024)
	tests := []struct {
		name        string
		apiKey      string
		model       string
		wantErr     bool
		wantErrText string
	}{
		{
			name:        "whitespace only key rejected",
			apiKey:      "   \n\t   ",
			model:       "openrouter/model",
			wantErr:     true,
			wantErrText: "openrouter api key cannot be empty",
		},
		{
			name:    "null byte key accepted as non-empty",
			apiKey:  "sk-live\x00evil",
			model:   "openrouter/model",
			wantErr: false,
		},
		{
			name:    "extremely long key accepted without panic",
			apiKey:  veryLong,
			model:   "openrouter/model",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, err := newOpenRouterProvider(&config.Config{
				OpenRouterAPIKey: tt.apiKey,
				OpenRouterModel:  tt.model,
			})

			if tt.wantErr {
				if provider != nil {
					t.Fatalf("provider = %T, want nil", provider)
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.wantErrText {
					t.Fatalf("error = %q, want %q", err.Error(), tt.wantErrText)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider == nil {
				t.Fatal("provider is nil, want non-nil")
			}
		})
	}
}

func TestAdversarialModelNameInjection(t *testing.T) {
	t.Parallel()

	t.Run("special characters and traversal patterns are handled", func(t *testing.T) {
		t.Parallel()

		model := "../../etc/passwd?x=${jndi:ldap://evil}#<script>alert(1)</script>"
		provider, err := newOpenRouterProvider(&config.Config{
			OpenRouterAPIKey: "safe-key",
			OpenRouterModel:  model,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if provider.model != model {
			t.Fatalf("model = %q, want %q", provider.model, model)
		}
	})

	t.Run("empty model rejected", func(t *testing.T) {
		t.Parallel()

		provider, err := newOpenRouterProvider(&config.Config{
			OpenRouterAPIKey: "safe-key",
			OpenRouterModel:  "   \n\t",
		})
		if provider != nil {
			t.Fatalf("provider = %T, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "openrouter model cannot be empty" {
			t.Fatalf("error = %q, want %q", err.Error(), "openrouter model cannot be empty")
		}
	})
}

func TestAdversarialBuildStructureUserPromptInjection(t *testing.T) {
	t.Parallel()

	title := "# Title\n<script>alert(1)</script>\u202E"
	participants := []string{
		"Alice",
		"Bob, <img src=x onerror=alert(1)>",
		"Mallory${7*7}\x00",
	}
	raw := strings.Repeat("A", 11*1024) + "\n../etc/passwd\n<script>alert(1)</script>"

	got := buildStructureUserPrompt(raw, MeetingMeta{
		Title:        title,
		Date:         "2026-04-16",
		Participants: participants,
		TicketID:     "TCK-1",
	})

	if !strings.Contains(got, "Title: "+title) {
		t.Fatalf("prompt missing injected title; got %q", got)
	}
	if !strings.Contains(got, "Participants: Alice, Bob, <img src=x onerror=alert(1)>, Mallory${7*7}\x00") {
		t.Fatalf("prompt missing participant payloads; got %q", got)
	}
	if !strings.Contains(got, "Raw Notes:\n"+raw) {
		t.Fatalf("prompt missing raw notes payload; length=%d", len(got))
	}
	if len(got) <= 11*1024 {
		t.Fatalf("prompt length = %d, want > %d", len(got), 11*1024)
	}
}

func TestAdversarialRefererRoundTripperOverwritesHeader(t *testing.T) {
	t.Parallel()

	capturing := &adversarialCaptureTransport{}
	rt := &refererRoundTripper{base: capturing, referer: "github.com/glieske/recap"}

	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	req.Header.Add("HTTP-Referer", "attacker.example")
	req.Header.Add("HTTP-Referer", "second.attacker")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	_ = resp.Body.Close()

	if capturing.lastRequest == nil {
		t.Fatal("capturing transport did not receive request")
	}

	values := capturing.lastRequest.Header.Values("HTTP-Referer")
	if len(values) != 1 {
		t.Fatalf("HTTP-Referer values len = %d, want 1", len(values))
	}
	if values[0] != "github.com/glieske/recap" {
		t.Fatalf("HTTP-Referer = %q, want %q", values[0], "github.com/glieske/recap")
	}
}

func TestAdversarialNewProviderMaliciousProviderName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		aiProvider string
	}{
		{name: "injection chars", aiProvider: "openrouter\n\x00;rm -rf /"},
		{name: "empty provider", aiProvider: ""},
		{name: "whitespace only provider", aiProvider: "   \n\t  "},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, err := NewProvider(&config.Config{
				AIProvider:       tt.aiProvider,
				GitHubModel:      "", // force github_models fail
				OpenRouterAPIKey: "", // force openrouter fail
				OpenRouterModel:  "",
				LMStudioURL:      "", // lm_studio should succeed with default URL
				LMStudioModel:    "local-model",
			})
			if err != nil {
				t.Fatalf("NewProvider returned error: %v", err)
			}
			if _, ok := provider.(*LMStudioProvider); !ok {
				t.Fatalf("provider type = %T, want *LMStudioProvider", provider)
			}
		})
	}
}

func TestAdversarialConstructProviderNilConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		providerName string
		wantErr      string
	}{
		{providerName: "github_models", wantErr: "config cannot be nil"},
		{providerName: "openrouter", wantErr: "config cannot be nil"},
		{providerName: "lm_studio", wantErr: "config cannot be nil"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.providerName, func(t *testing.T) {
			t.Parallel()

			provider, err := constructProvider(tt.providerName, nil)
			if provider != nil {
				t.Fatalf("provider = %T, want nil", provider)
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAdversarialChatCompletionEmptyMessagesNoPanic(t *testing.T) {
	t.Parallel()

	got, err := chatCompletion(context.Background(), nil, "model", []openai.ChatCompletionMessage{}, 0.2)
	if got != "" {
		t.Fatalf("result = %q, want empty string", got)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "openai client cannot be nil" {
		t.Fatalf("error = %q, want %q", err.Error(), "openai client cannot be nil")
	}
}

func TestAdversarialProviderMethodsNoPanicMaliciousPayloads(t *testing.T) {
	t.Parallel()

	maliciousMeta := MeetingMeta{
		Title:        "<script>alert(1)</script>\u202E",
		Date:         "2026-04-16",
		Participants: []string{"../etc/passwd", "Mallory\x00", strings.Repeat("X", 2048)},
		TicketID:     "${jndi:ldap://evil}",
	}
	raw := strings.Repeat("notes\n", 3000)
	structured := "# Summary\n" + strings.Repeat("<img src=x onerror=alert(1)>\n", 500)

	t.Run("github models methods", func(t *testing.T) {
		t.Parallel()

		provider := &GitHubModelsProvider{client: nil, model: "../../model"}

		got1, err1 := provider.StructureNotes(context.Background(), raw, maliciousMeta)
		if got1 != "" {
			t.Fatalf("StructureNotes result = %q, want empty string", got1)
		}
		if err1 == nil {
			t.Fatal("StructureNotes expected error, got nil")
		}
		if err1.Error() != "openai client cannot be nil" {
			t.Fatalf("StructureNotes error = %q, want %q", err1.Error(), "openai client cannot be nil")
		}

		got2, err2 := provider.GenerateEmailSummary(context.Background(), structured, "en")
		if got2 != "" {
			t.Fatalf("GenerateEmailSummary result = %q, want empty string", got2)
		}
		if err2 == nil {
			t.Fatal("GenerateEmailSummary expected error, got nil")
		}
		if err2.Error() != "openai client cannot be nil" {
			t.Fatalf("GenerateEmailSummary error = %q, want %q", err2.Error(), "openai client cannot be nil")
		}
	})

	t.Run("openrouter methods", func(t *testing.T) {
		t.Parallel()

		provider := &OpenRouterProvider{client: nil, model: "..//openrouter?model"}

		got1, err1 := provider.StructureNotes(context.Background(), raw, maliciousMeta)
		if got1 != "" {
			t.Fatalf("StructureNotes result = %q, want empty string", got1)
		}
		if err1 == nil {
			t.Fatal("StructureNotes expected error, got nil")
		}
		if err1.Error() != "openai client cannot be nil" {
			t.Fatalf("StructureNotes error = %q, want %q", err1.Error(), "openai client cannot be nil")
		}

		got2, err2 := provider.GenerateEmailSummary(context.Background(), structured, "en")
		if got2 != "" {
			t.Fatalf("GenerateEmailSummary result = %q, want empty string", got2)
		}
		if err2 == nil {
			t.Fatal("GenerateEmailSummary expected error, got nil")
		}
		if err2.Error() != "openai client cannot be nil" {
			t.Fatalf("GenerateEmailSummary error = %q, want %q", err2.Error(), "openai client cannot be nil")
		}
	})

	t.Run("lm studio methods", func(t *testing.T) {
		t.Parallel()

		provider := &LMStudioProvider{client: nil, model: ""}

		got1, err1 := provider.StructureNotes(context.Background(), raw, maliciousMeta)
		if got1 != "" {
			t.Fatalf("StructureNotes result = %q, want empty string", got1)
		}
		if err1 == nil {
			t.Fatal("StructureNotes expected error, got nil")
		}
		if err1.Error() != "openai client cannot be nil" {
			t.Fatalf("StructureNotes error = %q, want %q", err1.Error(), "openai client cannot be nil")
		}

		got2, err2 := provider.GenerateEmailSummary(context.Background(), structured, "en")
		if got2 != "" {
			t.Fatalf("GenerateEmailSummary result = %q, want empty string", got2)
		}
		if err2 == nil {
			t.Fatal("GenerateEmailSummary expected error, got nil")
		}
		if err2.Error() != "openai client cannot be nil" {
			t.Fatalf("GenerateEmailSummary error = %q, want %q", err2.Error(), "openai client cannot be nil")
		}
	})
}

type adversarialCaptureTransport struct {
	lastRequest *http.Request
}

func (a *adversarialCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	a.lastRequest = req
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}
