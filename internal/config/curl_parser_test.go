package config

import (
	"testing"
)

func TestParseCURL(t *testing.T) {
	testCases := []struct {
		name         string
		cmd          string
		wantEndpoint string
		wantModel    string
		wantToken    string
	}{
		{
			name:         "NVIDIA Integrate with single quotes",
			cmd:          `curl https://integrate.api.nvidia.com/v1/ -H 'Content-Type: application/json' -H 'Authorization: Bearer nvapi-123456789' -d '{"model": "nvidia/nemotron-3-ultra-550b-a55b"}'`,
			wantEndpoint: "https://integrate.api.nvidia.com/v1/chat/completions",
			wantModel:    "nvidia/nemotron-3-ultra-550b-a55b",
			wantToken:    "nvapi-123456789",
		},
		{
			name:         "NVIDIA with backslash escaped quotes",
			cmd:          `curl https://integrate.api.nvidia.com/v1/chat/completions -H "Authorization: Bearer nvapi-987" -d "{\"model\": \"nvidia/nemotron-3-ultra-550b-a55b\"}"`,
			wantEndpoint: "https://integrate.api.nvidia.com/v1/chat/completions",
			wantModel:    "nvidia/nemotron-3-ultra-550b-a55b",
			wantToken:    "nvapi-987",
		},
		{
			name:         "User raw syntax with escaped quotes inside single quotes",
			cmd:          `curl https://integrate.api.nvidia.com/v1/ -H 'Content-Type: application/json' -H 'Authorization: Bearer nvapi-abc' -d '{\"model\": \"nvidia/nemotron-3-ultra-550b-a55b"}'"`,
			wantEndpoint: "https://integrate.api.nvidia.com/v1/chat/completions",
			wantModel:    "nvidia/nemotron-3-ultra-550b-a55b",
			wantToken:    "nvapi-abc",
		},
		{
			name:         "OpenRouter endpoint without /chat/completions",
			cmd:          `curl https://openrouter.ai/api/v1 -H "Authorization: Bearer sk-or-v1-123" -d '{"model": "anthropic/claude-3.5-sonnet"}'`,
			wantEndpoint: "https://openrouter.ai/api/v1/chat/completions",
			wantModel:    "anthropic/claude-3.5-sonnet",
			wantToken:    "sk-or-v1-123",
		},
		{
			name:         "Groq with auto fallback model",
			cmd:          `curl https://api.groq.com/openai/v1/chat/completions -H "Authorization: Bearer gsk_123"`,
			wantEndpoint: "https://api.groq.com/openai/v1/chat/completions",
			wantModel:    "llama-3.3-70b-versatile",
			wantToken:    "gsk_123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseCURL(tc.cmd)
			if err != nil {
				t.Fatalf("ParseCURL failed: %v", err)
			}
			if cfg.Endpoint != tc.wantEndpoint {
				t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, tc.wantEndpoint)
			}
			if cfg.Model != tc.wantModel {
				t.Errorf("Model = %q, want %q", cfg.Model, tc.wantModel)
			}
			if cfg.APIKey != tc.wantToken {
				t.Errorf("APIKey = %q, want %q", cfg.APIKey, tc.wantToken)
			}
		})
	}
}
