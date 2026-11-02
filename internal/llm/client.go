package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lucinexus/lucinexus/internal/config"
	"github.com/lucinexus/lucinexus/internal/tools"
)

type LLMClient interface {
	Plan(ctx context.Context, systemPrompt, userQuery string, availableTools []tools.ToolDef) ([]tools.ToolCall, string, error)
}

func NewClient(cfg *config.Config) (LLMClient, error) {
	switch cfg.Provider {
	case config.ProviderGemini:
		return &geminiClient{cfg: cfg, httpClient: &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}}, nil
	case config.ProviderOpenAI, config.ProviderLocal, config.ProviderCustom:
		return &customOpenAIClient{cfg: cfg, httpClient: &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

type customOpenAIClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

func (c *customOpenAIClient) Plan(ctx context.Context, systemPrompt, userQuery string, availableTools []tools.ToolDef) ([]tools.ToolCall, string, error) {
	ep := "https://api.openai.com/v1/chat/completions"
	if c.cfg.Endpoint != "" {
		ep = c.cfg.Endpoint
	}

	payload := map[string]interface{}{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userQuery},
		},
	}

	if len(availableTools) > 0 {
		var openAITools []map[string]interface{}
		for _, t := range availableTools {
			openAITools = append(openAITools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			})
		}
		payload["tools"] = openAITools
		payload["tool_choice"] = "auto"
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", ep, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	if c.cfg.APIKey != "" {
		hName := c.cfg.AuthHeader
		if hName == "" {
			hName = "Authorization"
		}
		req.Header.Set(hName, c.cfg.AuthPrefix+c.cfg.APIKey)
	}

	for k, v := range c.cfg.CustomHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	resData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(resData))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	_ = json.Unmarshal(resData, &res)
	if len(res.Choices) == 0 {
		return nil, "", fmt.Errorf("empty response choices from provider")
	}

	msg := res.Choices[0].Message
	var parsedCalls []tools.ToolCall
	for _, tc := range msg.ToolCalls {
		parsedCalls = append(parsedCalls, tools.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	return parsedCalls, msg.Content, nil
}

type geminiClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

func (g *geminiClient) Plan(ctx context.Context, systemPrompt, userQuery string, availableTools []tools.ToolDef) ([]tools.ToolCall, string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.cfg.Model, g.cfg.APIKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": systemPrompt + "\nUser Request: " + userQuery},
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	resData, _ := io.ReadAll(resp.Body)
	return nil, string(resData), nil
}
