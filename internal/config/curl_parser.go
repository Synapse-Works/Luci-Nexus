package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CustomProviderConfig struct {
	Name          string            `json:"name"`
	Endpoint      string            `json:"endpoint"`
	Model         string            `json:"model"`
	APIKey        string            `json:"api_key"`
	AuthHeader    string            `json:"auth_header"`
	AuthPrefix    string            `json:"auth_prefix"`
	Protocol      string            `json:"protocol"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

func ParseCURL(curlCmd string) (*CustomProviderConfig, error) {
	clean := strings.ReplaceAll(curlCmd, "\
", " ")
	clean = strings.ReplaceAll(clean, "\
", " ")
	clean = strings.TrimSpace(clean)

	tokens := tokenizeShellArgs(clean)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty curl command")
	}

	cfg := &CustomProviderConfig{
		Name:          "custom-curl",
		AuthHeader:    "Authorization",
		AuthPrefix:    "Bearer ",
		Protocol:      "openai-compatible",
		CustomHeaders: make(map[string]string),
	}

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]

		if (t == "-H" || t == "--header") && i+1 < len(tokens) {
			headerVal := tokens[i+1]
			i++
			parts := strings.SplitN(headerVal, ":", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])

				if strings.EqualFold(k, "authorization") {
					cfg.AuthHeader = "Authorization"
					if strings.HasPrefix(strings.ToLower(v), "bearer ") {
						cfg.AuthPrefix = "Bearer "
						cfg.APIKey = strings.TrimSpace(v[7:])
					} else {
						cfg.AuthPrefix = ""
						cfg.APIKey = v
					}
				} else if strings.EqualFold(k, "api-key") || strings.EqualFold(k, "x-api-key") {
					cfg.AuthHeader = k
					cfg.AuthPrefix = ""
					cfg.APIKey = v
				} else if !strings.EqualFold(k, "content-type") {
					cfg.CustomHeaders[k] = v
				}
			}
			continue
		}

		if (t == "-d" || t == "--data" || t == "--data-raw" || t == "--data-binary") && i+1 < len(tokens) {
			dataPayload := tokens[i+1]
			i++
			var bodyMap map[string]interface{}
			if err := json.Unmarshal([]byte(dataPayload), &bodyMap); err == nil {
				if m, ok := bodyMap["model"].(string); ok && m != "" {
					cfg.Model = m
				}
			}
			continue
		}

		if t == "--url" && i+1 < len(tokens) {
			cfg.Endpoint = tokens[i+1]
			i++
			continue
		}
		if (strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://")) && cfg.Endpoint == "" {
			cfg.Endpoint = t
		}
	}

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("could not detect target HTTP(S) URL in curl command")
	}

	if _, err := url.ParseRequestURI(cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL '%s': %w", cfg.Endpoint, err)
	}

	if cfg.Model == "" {
		if strings.Contains(cfg.Endpoint, "deepseek") {
			cfg.Model = "deepseek-chat"
		} else if strings.Contains(cfg.Endpoint, "groq") {
			cfg.Model = "llama-3.3-70b-versatile"
		} else {
			cfg.Model = "default"
		}
	}

	return cfg, nil
}

func (c *CustomProviderConfig) VerifyEndpoint(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	probeBody, _ := json.Marshal(map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewReader(probeBody))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set(c.AuthHeader, c.AuthPrefix+c.APIKey)
	}
	for k, v := range c.CustomHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("handshake failed: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return latency, fmt.Errorf("authentication error (HTTP %d): verify API key", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		respBytes, _ := io.ReadAll(resp.Body)
		return latency, fmt.Errorf("remote server error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	return latency, nil
}

func tokenizeShellArgs(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		r := cmd[i]
		if r == ''' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (r == ' ' || r == '	') && !inSingle && !inDouble {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}
