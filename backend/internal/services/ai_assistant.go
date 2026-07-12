package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

)

// AIAssistantConfig holds configuration for the AI assistant.
type AIAssistantConfig struct {
	Enabled  bool
	Provider string // "openai", "ollama"
	APIKey   string
	Model    string
	Endpoint string // custom endpoint for Ollama or OpenAI-compatible
}

// AIAssistantResponse represents the AI's response.
type AIAssistantResponse struct {
	Response  string `json:"response"`
	Model     string `json:"model"`
	Usage     *AIUsage `json:"usage,omitempty"`
}

// AIUsage tracks token usage.
type AIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AIAssistantService provides AI-powered code assistance.
type AIAssistantService struct {
	cfg AIAssistantConfig
}

// NewAIAssistantService creates a new AI assistant service.
func NewAIAssistantService(cfg AIAssistantConfig) *AIAssistantService {
	return &AIAssistantService{cfg: cfg}
}

// IsEnabled reports whether the AI assistant is configured.
func (s *AIAssistantService) IsEnabled() bool {
	return s.cfg.Enabled && s.cfg.Provider != ""
}

// Ask sends a prompt to the AI provider and returns the response.
func (s *AIAssistantService) Ask(ctx context.Context, prompt, notebookID, codeContext string) (*AIAssistantResponse, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("AI assistant is not enabled")
	}

	systemPrompt := "You are a data engineering AI assistant integrated into a Spark notebook platform (RCN). " +
		"Help the user write Spark/Python/SQL code. Provide concise, production-quality solutions."

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}
	if codeContext != "" {
		messages = append(messages, map[string]string{
			"role":    "user",
			"content": fmt.Sprintf("Here is my current notebook code context:\n```python\n%s\n```\n\n", codeContext),
		})
	}
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": prompt,
	})

	switch s.cfg.Provider {
	case "openai":
		return s.callOpenAI(ctx, messages)
	case "ollama":
		return s.callOllama(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", s.cfg.Provider)
	}
}

func (s *AIAssistantService) callOpenAI(ctx context.Context, messages []map[string]string) (*AIAssistantResponse, error) {
	body := map[string]interface{}{
		"model":       s.cfg.Model,
		"messages":    messages,
		"max_tokens":  2048,
		"temperature": 0.3,
	}

	jsonBody, _ := json.Marshal(body)
	endpoint := s.cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *AIUsage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from OpenAI")
	}

	return &AIAssistantResponse{
		Response: result.Choices[0].Message.Content,
		Model:    s.cfg.Model,
		Usage:    result.Usage,
	}, nil
}

func (s *AIAssistantService) callOllama(ctx context.Context, prompt string) (*AIAssistantResponse, error) {
	endpoint := s.cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	body := map[string]interface{}{
		"model":  s.cfg.Model,
		"prompt": prompt,
		"stream": false,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w", err)
	}

	return &AIAssistantResponse{
		Response: result.Response,
		Model:    s.cfg.Model,
	}, nil
}
