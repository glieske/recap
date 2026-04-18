package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"unsafe"

	"github.com/glieske/recap/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

func TestBuildProviderOrder_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		want       []string
	}{
		{
			name:       "configured github_models",
			configured: "github_models",
			want:       []string{"github_models", "openrouter", "lm_studio"},
		},
		{
			name:       "configured lm_studio",
			configured: "lm_studio",
			want:       []string{"lm_studio", "github_models", "openrouter"},
		},
		{
			name:       "empty configured uses default",
			configured: "",
			want:       []string{"github_models", "openrouter", "lm_studio"},
		},
		{
			name:       "configured openrouter",
			configured: "openrouter",
			want:       []string{"openrouter", "github_models", "lm_studio"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildProviderOrder(tt.configured)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildProviderOrder(%q) = %v, want %v", tt.configured, got, tt.want)
			}
		})
	}
}

func TestBuildProviderOrder_Property(t *testing.T) {
	t.Parallel()

	defaultOrder := []string{"github_models", "openrouter", "lm_studio"}

	prop := func(in string) bool {
		got := buildProviderOrder(in)
		trimmed := strings.TrimSpace(in)

		if trimmed == "" {
			return reflect.DeepEqual(got, defaultOrder)
		}

		if len(got) == 0 || got[0] != trimmed {
			return false
		}

		positions := make(map[string]int, len(defaultOrder))
		for i, p := range got {
			if _, ok := positions[p]; !ok {
				positions[p] = i
			}
		}

		for _, p := range defaultOrder {
			if _, ok := positions[p]; !ok {
				return false
			}
		}

		return positions["github_models"] < positions["openrouter"] && positions["openrouter"] < positions["lm_studio"]
	}

	if err := quick.Check(prop, nil); err != nil {
		t.Fatalf("property check failed: %v", err)
	}
}

func TestBuildStructureUserPrompt(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()

		meta := MeetingMeta{
			Title:        "Sprint Planning",
			Date:         "2026-04-16",
			Participants: []string{"Alice", "Bob"},
			TicketID:     "PROJ-123",
		}

		got := buildStructureUserPrompt("Raw bullet points", meta)
		want := "Title: Sprint Planning\nDate: 2026-04-16\nParticipants: Alice, Bob\nTicket ID: PROJ-123\n\nRaw Notes:\nRaw bullet points"

		if got != want {
			t.Fatalf("unexpected prompt\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("empty participants", func(t *testing.T) {
		t.Parallel()

		meta := MeetingMeta{Title: "Retro", Date: "2026-04-16", Participants: nil, TicketID: "PROJ-1"}

		got := buildStructureUserPrompt("notes", meta)
		want := "Title: Retro\nDate: 2026-04-16\nParticipants: \nTicket ID: PROJ-1\n\nRaw Notes:\nnotes"

		if got != want {
			t.Fatalf("unexpected prompt for empty participants\n got: %q\nwant: %q", got, want)
		}
	})

	t.Run("empty ticket id", func(t *testing.T) {
		t.Parallel()

		meta := MeetingMeta{Title: "Standup", Date: "2026-04-16", Participants: []string{"Eve"}, TicketID: ""}

		got := buildStructureUserPrompt("items", meta)
		want := "Title: Standup\nDate: 2026-04-16\nParticipants: Eve\nTicket ID: \n\nRaw Notes:\nitems"

		if got != want {
			t.Fatalf("unexpected prompt for empty ticket id\n got: %q\nwant: %q", got, want)
		}
	})
}

func TestConstructProvider(t *testing.T) {
	t.Parallel()

	t.Run("lm_studio valid config returns LMStudioProvider", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{LMStudioURL: "http://localhost:1234/v1", LMStudioModel: "local-model"}
		provider, err := constructProvider("lm_studio", cfg)
		if err != nil {
			t.Fatalf("constructProvider returned error: %v", err)
		}

		if _, ok := provider.(*LMStudioProvider); !ok {
			t.Fatalf("provider type = %T, want *LMStudioProvider", provider)
		}
	})

	t.Run("openrouter empty api key returns error", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{OpenRouterAPIKey: "", OpenRouterModel: "openrouter/model"}
		provider, err := constructProvider("openrouter", cfg)
		if provider != nil {
			t.Fatalf("provider = %T, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "openrouter api key cannot be empty" {
			t.Fatalf("error = %q, want %q", err.Error(), "openrouter api key cannot be empty")
		}
	})

	t.Run("unknown provider returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := constructProvider("unknown", &config.Config{})
		if provider != nil {
			t.Fatalf("provider = %T, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "unsupported provider: unknown" {
			t.Fatalf("error = %q, want %q", err.Error(), "unsupported provider: unknown")
		}
	})

	t.Run("github_models conditional", func(t *testing.T) {
		t.Parallel()

		if _, err := exec.LookPath("gh"); err != nil {
			t.Skip("gh CLI not available")
		}

		provider, err := constructProvider("github_models", &config.Config{GitHubModel: "gpt-4o"})
		if err == nil {
			if _, ok := provider.(*GitHubModelsProvider); !ok {
				t.Fatalf("provider type = %T, want *GitHubModelsProvider", provider)
			}
			return
		}

		if !strings.Contains(err.Error(), "failed to retrieve github auth token") && !strings.Contains(err.Error(), "github auth token is empty") {
			t.Fatalf("unexpected github_models error: %v", err)
		}
	})
}

func TestNewLMStudioProvider(t *testing.T) {
	t.Parallel()

	t.Run("empty lm studio url uses default", func(t *testing.T) {
		t.Parallel()

		provider, err := newLMStudioProvider(&config.Config{LMStudioURL: "", LMStudioModel: "local"})
		if err != nil {
			t.Fatalf("newLMStudioProvider returned error: %v", err)
		}

		cfg := extractOpenAIClientConfig(t, provider.client)
		if cfg.BaseURL != defaultLMStudioURL {
			t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, defaultLMStudioURL)
		}
	})

	t.Run("custom lm studio url is used", func(t *testing.T) {
		t.Parallel()

		custom := "http://127.0.0.1:5678/v1"
		provider, err := newLMStudioProvider(&config.Config{LMStudioURL: custom, LMStudioModel: "local"})
		if err != nil {
			t.Fatalf("newLMStudioProvider returned error: %v", err)
		}

		cfg := extractOpenAIClientConfig(t, provider.client)
		if cfg.BaseURL != custom {
			t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, custom)
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := newLMStudioProvider(nil)
		if provider != nil {
			t.Fatalf("provider = %v, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "config cannot be nil" {
			t.Fatalf("error = %q, want %q", err.Error(), "config cannot be nil")
		}
	})
}

func TestNewOpenRouterProvider(t *testing.T) {
	t.Parallel()

	t.Run("empty api key returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := newOpenRouterProvider(&config.Config{OpenRouterAPIKey: "", OpenRouterModel: "some/model"})
		if provider != nil {
			t.Fatalf("provider = %v, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "openrouter api key cannot be empty" {
			t.Fatalf("error = %q, want %q", err.Error(), "openrouter api key cannot be empty")
		}
	})

	t.Run("empty model returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := newOpenRouterProvider(&config.Config{OpenRouterAPIKey: "key", OpenRouterModel: ""})
		if provider != nil {
			t.Fatalf("provider = %v, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "openrouter model cannot be empty" {
			t.Fatalf("error = %q, want %q", err.Error(), "openrouter model cannot be empty")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := newOpenRouterProvider(nil)
		if provider != nil {
			t.Fatalf("provider = %v, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "config cannot be nil" {
			t.Fatalf("error = %q, want %q", err.Error(), "config cannot be nil")
		}
	})
}

func TestNewProvider(t *testing.T) {
	t.Parallel()

	t.Run("nil config returns error", func(t *testing.T) {
		t.Parallel()

		provider, err := NewProvider(nil)
		if provider != nil {
			t.Fatalf("provider = %T, want nil", provider)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "config cannot be nil" {
			t.Fatalf("error = %q, want %q", err.Error(), "config cannot be nil")
		}
	})

	t.Run("fallback reaches lm studio when others fail", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			AIProvider:       "openrouter",
			OpenRouterAPIKey: "",
			OpenRouterModel:  "",
			GitHubModel:      "",
			LMStudioURL:      "",
			LMStudioModel:    "local",
		}

		provider, err := NewProvider(cfg)
		if err != nil {
			t.Fatalf("NewProvider returned error: %v", err)
		}

		if _, ok := provider.(*LMStudioProvider); !ok {
			t.Fatalf("provider type = %T, want *LMStudioProvider", provider)
		}
	})
}

func TestChatCompletion_NilClient(t *testing.T) {
	t.Parallel()

	got, err := chatCompletion(context.Background(), nil, "model", nil, 0.2)
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

func TestProviderMethods_NilClientError(t *testing.T) {
	t.Parallel()

	meta := MeetingMeta{Title: "T", Date: "2026-04-16", Participants: []string{"A"}, TicketID: "X-1"}

	t.Run("github models structure notes", func(t *testing.T) {
		t.Parallel()
		provider := &GitHubModelsProvider{client: nil, model: "m"}
		got, err := provider.StructureNotes(context.Background(), "raw", meta)
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})

	t.Run("github models generate email summary", func(t *testing.T) {
		t.Parallel()
		provider := &GitHubModelsProvider{client: nil, model: "m"}
		got, err := provider.GenerateEmailSummary(context.Background(), "# structured", "en")
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})

	t.Run("openrouter structure notes", func(t *testing.T) {
		t.Parallel()
		provider := &OpenRouterProvider{client: nil, model: "m"}
		got, err := provider.StructureNotes(context.Background(), "raw", meta)
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})

	t.Run("openrouter generate email summary", func(t *testing.T) {
		t.Parallel()
		provider := &OpenRouterProvider{client: nil, model: "m"}
		got, err := provider.GenerateEmailSummary(context.Background(), "# structured", "en")
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})

	t.Run("lm studio structure notes", func(t *testing.T) {
		t.Parallel()
		provider := &LMStudioProvider{client: nil, model: "m"}
		got, err := provider.StructureNotes(context.Background(), "raw", meta)
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})

	t.Run("lm studio generate email summary", func(t *testing.T) {
		t.Parallel()
		provider := &LMStudioProvider{client: nil, model: "m"}
		got, err := provider.GenerateEmailSummary(context.Background(), "# structured", "en")
		if got != "" {
			t.Fatalf("result = %q, want empty string", got)
		}
		if err == nil || err.Error() != "openai client cannot be nil" {
			t.Fatalf("error = %v, want %q", err, "openai client cannot be nil")
		}
	})
}

func TestRefererRoundTripper(t *testing.T) {
	t.Parallel()

	t.Run("adds HTTP-Referer header", func(t *testing.T) {
		t.Parallel()

		capturing := &captureTransport{}
		rt := &refererRoundTripper{base: capturing, referer: "github.com/glieske/recap"}

		req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
		if err != nil {
			t.Fatalf("NewRequest error: %v", err)
		}
		req.Header.Set("X-Test", "value")

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip error: %v", err)
		}
		_ = resp.Body.Close()

		if capturing.lastRequest == nil {
			t.Fatal("capturing transport did not receive request")
		}
		if got := capturing.lastRequest.Header.Get("HTTP-Referer"); got != "github.com/glieske/recap" {
			t.Fatalf("HTTP-Referer = %q, want %q", got, "github.com/glieske/recap")
		}

		if got := req.Header.Get("HTTP-Referer"); got != "" {
			t.Fatalf("original request header mutated: HTTP-Referer = %q, want empty", got)
		}

		if got := capturing.lastRequest.Header.Get("X-Test"); got != "value" {
			t.Fatalf("X-Test = %q, want %q", got, "value")
		}
	})

	t.Run("nil base transport falls back to default transport", func(t *testing.T) {
		t.Parallel()

		seenReferer := ""
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenReferer = r.Header.Get("HTTP-Referer")
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		rt := &refererRoundTripper{base: nil, referer: "github.com/glieske/recap"}

		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest error: %v", err)
		}

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip error: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusNoContent)
		}

		if seenReferer != "github.com/glieske/recap" {
			t.Fatalf("server saw referer = %q, want %q", seenReferer, "github.com/glieske/recap")
		}
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()

		rt := &refererRoundTripper{base: http.DefaultTransport, referer: "github.com/glieske/recap"}
		resp, err := rt.RoundTrip(nil)
		if resp != nil {
			t.Fatalf("response = %v, want nil", resp)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "request cannot be nil" {
			t.Fatalf("error = %q, want %q", err.Error(), "request cannot be nil")
		}
	})
}

type captureTransport struct {
	lastRequest *http.Request
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.lastRequest = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func extractOpenAIClientConfig(t *testing.T, client *openai.Client) openai.ClientConfig {
	t.Helper()

	if client == nil {
		t.Fatal("client cannot be nil")
	}

	v := reflect.ValueOf(client).Elem()
	configField := v.FieldByName("config")
	if !configField.IsValid() {
		t.Fatal("openai.Client has no config field")
	}

	cfg := reflect.NewAt(configField.Type(), unsafe.Pointer(configField.UnsafeAddr())).Elem().Interface().(openai.ClientConfig)
	return cfg
}
