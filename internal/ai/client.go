package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/glieske/recap/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

const (
	githubModelsBaseURL = "https://models.inference.ai.azure.com"
	openRouterBaseURL   = "https://openrouter.ai/api/v1"
	defaultLMStudioURL  = "http://localhost:1234/v1"
	openRouterReferer   = "github.com/glieske/recap"
)

// MeetingMeta contains metadata used during note structuring.
type MeetingMeta struct {
	Title        string
	Date         string
	Participants []string
	TicketID     string
}

// Provider defines AI operations supported by the application.
type Provider interface {
	StructureNotes(ctx context.Context, rawNotes string, meta MeetingMeta) (string, error)
	GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error)
}

type GitHubModelsProvider struct {
	client *openai.Client
	model  string
}

type OpenRouterProvider struct {
	client *openai.Client
	model  string
}

type LMStudioProvider struct {
	client *openai.Client
	model  string
}

func NewProvider(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	orderedProviders := buildProviderOrder(cfg.AIProvider)
	attemptErrors := make([]string, 0, len(orderedProviders))

	for _, providerName := range orderedProviders {
		provider, err := constructProvider(providerName, cfg)
		if err == nil {
			return provider, nil
		}

		attemptErrors = append(attemptErrors, fmt.Sprintf("%s: %v", providerName, err))
	}

	return nil, fmt.Errorf("failed to initialize AI provider: %s", strings.Join(attemptErrors, "; "))
}

func newGitHubModelsProvider(cfg *config.Config) (*GitHubModelsProvider, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	if strings.TrimSpace(cfg.GitHubModel) == "" {
		return nil, errors.New("github model cannot be empty")
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("github cli not found in PATH: %w", err)
	}

	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("failed to retrieve github auth token via 'gh auth token': %w (%s)", err, strings.TrimSpace(string(exitErr.Stderr)))
		}

		return nil, fmt.Errorf("failed to retrieve github auth token via 'gh auth token': %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return nil, errors.New("github auth token is empty")
	}

	clientConfig := openai.DefaultConfig(token)
	clientConfig.BaseURL = githubModelsBaseURL

	return &GitHubModelsProvider{
		client: openai.NewClientWithConfig(clientConfig),
		model:  cfg.GitHubModel,
	}, nil
}

func newOpenRouterProvider(cfg *config.Config) (*OpenRouterProvider, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	apiKey := strings.TrimSpace(cfg.OpenRouterAPIKey)
	if apiKey == "" {
		return nil, errors.New("openrouter api key cannot be empty")
	}

	if strings.TrimSpace(cfg.OpenRouterModel) == "" {
		return nil, errors.New("openrouter model cannot be empty")
	}

	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = openRouterBaseURL
	clientConfig.HTTPClient = &http.Client{
		Transport: &refererRoundTripper{
			base:    http.DefaultTransport,
			referer: openRouterReferer,
		},
	}

	return &OpenRouterProvider{
		client: openai.NewClientWithConfig(clientConfig),
		model:  cfg.OpenRouterModel,
	}, nil
}

func newLMStudioProvider(cfg *config.Config) (*LMStudioProvider, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	baseURL := strings.TrimSpace(cfg.LMStudioURL)
	if baseURL == "" {
		baseURL = defaultLMStudioURL
	}

	clientConfig := openai.DefaultConfig("")
	clientConfig.BaseURL = baseURL

	return &LMStudioProvider{
		client: openai.NewClientWithConfig(clientConfig),
		model:  strings.TrimSpace(cfg.LMStudioModel),
	}, nil
}

func (p *GitHubModelsProvider) StructureNotes(ctx context.Context, rawNotes string, meta MeetingMeta) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a meeting notes assistant. Structure the following raw notes into organized Markdown.",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: buildStructureUserPrompt(rawNotes, meta),
		},
	}

	return chatCompletion(ctx, p.client, p.model, messages, 0.2)
}

func (p *GitHubModelsProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	messages := EmailPrompt(structuredMD, language)
	return chatCompletion(ctx, p.client, p.model, messages, EmailTemperature)
}

func (p *OpenRouterProvider) StructureNotes(ctx context.Context, rawNotes string, meta MeetingMeta) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a meeting notes assistant. Structure the following raw notes into organized Markdown.",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: buildStructureUserPrompt(rawNotes, meta),
		},
	}

	return chatCompletion(ctx, p.client, p.model, messages, 0.2)
}

func (p *OpenRouterProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	messages := EmailPrompt(structuredMD, language)
	return chatCompletion(ctx, p.client, p.model, messages, EmailTemperature)
}

func (p *LMStudioProvider) StructureNotes(ctx context.Context, rawNotes string, meta MeetingMeta) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a meeting notes assistant. Structure the following raw notes into organized Markdown.",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: buildStructureUserPrompt(rawNotes, meta),
		},
	}

	return chatCompletion(ctx, p.client, p.model, messages, 0.2)
}

func (p *LMStudioProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	messages := EmailPrompt(structuredMD, language)
	return chatCompletion(ctx, p.client, p.model, messages, EmailTemperature)
}

func chatCompletion(
	ctx context.Context,
	client *openai.Client,
	model string,
	messages []openai.ChatCompletionMessage,
	temperature float32,
) (string, error) {
	if client == nil {
		return "", errors.New("openai client cannot be nil")
	}

	response, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
	})
	if err != nil {
		return "", fmt.Errorf("create chat completion: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", errors.New("chat completion returned no choices")
	}

	return response.Choices[0].Message.Content, nil
}

func constructProvider(providerName string, cfg *config.Config) (Provider, error) {
	switch providerName {
	case "github_models":
		provider, err := newGitHubModelsProvider(cfg)
		if err != nil {
			return nil, err
		}

		return provider, nil
	case "openrouter":
		provider, err := newOpenRouterProvider(cfg)
		if err != nil {
			return nil, err
		}

		return provider, nil
	case "lm_studio":
		provider, err := newLMStudioProvider(cfg)
		if err != nil {
			return nil, err
		}

		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

func buildProviderOrder(configured string) []string {
	defaultOrder := []string{"github_models", "openrouter", "lm_studio"}
	normalizedConfigured := strings.TrimSpace(configured)
	if normalizedConfigured == "" {
		return defaultOrder
	}

	providerOrder := make([]string, 0, len(defaultOrder))
	providerOrder = append(providerOrder, normalizedConfigured)

	for _, providerName := range defaultOrder {
		if providerName == normalizedConfigured {
			continue
		}
		providerOrder = append(providerOrder, providerName)
	}

	return providerOrder
}

func buildStructureUserPrompt(rawNotes string, meta MeetingMeta) string {
	participants := ""
	if len(meta.Participants) > 0 {
		participants = strings.Join(meta.Participants, ", ")
	}

	return fmt.Sprintf(
		"Title: %s\nDate: %s\nParticipants: %s\nTicket ID: %s\n\nRaw Notes:\n%s",
		meta.Title,
		meta.Date,
		participants,
		meta.TicketID,
		rawNotes,
	)
}

type refererRoundTripper struct {
	base    http.RoundTripper
	referer string
}

func (r *refererRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	baseTransport := r.base
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	requestCopy := req.Clone(req.Context())
	requestCopy.Header = req.Header.Clone()
	requestCopy.Header.Set("HTTP-Referer", r.referer)

	return baseTransport.RoundTrip(requestCopy)
}
