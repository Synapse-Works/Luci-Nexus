# 🔌 LuciNexus: Custom AI Provider & cURL Import Specification

**Version:** 2.1.0

**Feature:** Manual Custom API Provider Configuration & Zero-Click cURL Import Engine

**Module:** `internal/config/curl_parser.go`, `internal/config/config.go`, `cmd/lucinexus/main.go`, `package/luci-app-lucinexus`

## 1. Overview & Architecture

Modern AI deployments span dozens of diverse providers beyond standard Google Gemini and OpenAI:

* **Cloud High-Performance APIs:** DeepSeek, Groq, Together AI, Mistral, OpenRouter, Cerebras, Perplexity.

* **Self-Hosted Local Homelab Endpoints:** Ollama, vLLM, LocalAI, LM Studio, text-generation-webui.

* **Custom Enterprise Proxies & Gateway Routers:** LiteLLM, Cloudflare AI Gateway, Portkey.

LuciNexus v2.1.0 introduces a dual-path configuration engine:

1. **Zero-Click cURL Importer:** Users can copy-paste any standard `curl` request snippet directly from API documentation or developer consoles. LuciNexus automatically parses arguments, extracts endpoint URLs, headers, auth tokens, and model identifiers, and verifies connectivity.

2. **Granular Manual Configuration:** Full UCI, CLI flag, and LuCI Web UI support for specifying custom endpoints, authentication header names (`Authorization`, `api-key`, `x-api-key`), token prefixes (`Bearer `, empty, custom), and protocol formats (`openai-compatible`, `ollama-native`, `anthropic`).

```
┌────────────────────────────────────────────────────────────────────────┐
│                        User Input Channels                             │
│    (LuCI Web UI Textarea / CLI Flag --curl / Interactive Wizard)       │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                     cURL Tokenizer & Parser                            │
│           (`internal/config/curl_parser.go:ParseCURL`)                  │
│   - Strips backslashes and normalizes multi-line shell snippets        │
│   - Tokenizes arguments (respects single and double quotes)            │
│   - Extracts URL, Headers (-H), Auth Tokens, and Body Data (-d)        │
│   - Detects Model ID and Protocol dialect                              │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                 Live Endpoint Handshake & Verification                 │
│         Sends lightweight ping to validate credentials and latency     │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                 OpenWrt UCI Persistence & Staging                      │
│        Commits to /etc/config/lucinexus with instant daemon reload     │
└────────────────────────────────────────────────────────────────────────┘

```

## 2. Go Implementation: cURL Tokenizer & Parser (`internal/config/curl_parser.go`)

This component robustly parses shell-escaped `curl` invocations without executing arbitrary shell code:

```
package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// CustomProviderConfig holds parameters for any arbitrary LLM provider.
type CustomProviderConfig struct {
	Name          string            `json:"name"`
	Endpoint      string            `json:"endpoint"`
	Model         string            `json:"model"`
	APIKey        string            `json:"api_key"`
	AuthHeader    string            `json:"auth_header"`    // e.g. "Authorization", "api-key"
	AuthPrefix    string            `json:"auth_prefix"`    // e.g. "Bearer ", ""
	Protocol      string            `json:"protocol"`       // "openai-compatible", "ollama", "anthropic"
	CustomHeaders map[string]string `json:"custom_headers"` // Additional headers (e.g. "HTTP-Referer")
}

// ParseCURL extracts API endpoint, headers, authorization tokens, and model name from a curl command string.
func ParseCURL(curlCmd string) (*CustomProviderConfig, error) {
	clean := strings.ReplaceAll(curlCmd, "\\\n", " ")
	clean = strings.ReplaceAll(clean, "\\\r\n", " ")
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

		// 1. Match Headers (-H or --header)
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
						cfg.APIKey = strings.TrimPrefix(v, v[:7])
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

		// 2. Match Payload Data (-d, --data, --data-raw)
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

		// 3. Match URL (bare argument starting with http:// or https://, or following --url)
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

	// Validate URL syntax
	if _, err := url.ParseRequestURI(cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL '%s': %w", cfg.Endpoint, err)
	}

	// Default model fallback if not explicitly in -d payload
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

// VerifyEndpoint tests latency and connectivity against the target custom API endpoint.
func (c *CustomProviderConfig) VerifyEndpoint(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	// Minimal probe payload
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

// tokenizeShellArgs splits a command string while respecting quotes and whitespace.
func tokenizeShellArgs(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		r := cmd[i]
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (r == ' ' || r == '\t') && !inSingle && !inDouble {
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

```

## 3. UCI Persistence & Dynamic Client Integration

### 3.1 UCI Storage Schema (`/etc/config/lucinexus`)

When configured, settings are written to the `lucinexus` UCI package:

```
config api 'custom'
	option enabled '1'
	option provider 'custom'
	option endpoint 'https://api.deepseek.com/chat/completions'
	option model 'deepseek-chat'
	option key 'sk-d7a8b9f0123456789abcdef'
	option auth_header 'Authorization'
	option auth_prefix 'Bearer '
	option protocol 'openai-compatible'
	option custom_headers 'X-Source:OpenWrt-Router'

```

### 3.2 Dynamic Multi-Provider Client Execution (`internal/llm/client.go`)

The LLM engine dynamically adapts its HTTP request structure based on the custom configuration:

```
// In internal/llm/client.go:
func (c *customClient) Plan(ctx context.Context, systemPrompt, userQuery string, availableTools []tools.ToolDef) ([]tools.ToolCall, string, error) {
	payload := map[string]interface{}{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userQuery},
		},
	}

	if len(availableTools) > 0 {
		var toolDefs []map[string]interface{}
		for _, t := range availableTools {
			toolDefs = append(toolDefs, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			})
		}
		payload["tools"] = toolDefs
		payload["tool_choice"] = "auto"
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		headerName := c.cfg.AuthHeader
		if headerName == "" {
			headerName = "Authorization"
		}
		req.Header.Set(headerName, c.cfg.AuthPrefix+c.cfg.APIKey)
	}

	// Apply custom enterprise headers
	for k, v := range c.cfg.CustomHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("custom provider request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response into structured tool calls...
	return parseOpenAICompatibleResponse(resp)
}

```

## 4. CLI Interface & Interactive Wizards

LuciNexus provides three methods for setting up custom AI providers from the router console:

### 4.1 Method A: Direct cURL Import via CLI

Copy any standard curl command from your provider's dashboard and pipe or pass it directly:

```
# Example 1: Import DeepSeek
lucinexus config --curl "curl https://api.deepseek.com/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-your-key-here' \
  -d '{\"model\": \"deepseek-chat\"}'"

# Example 2: Import Local Ollama Server on LAN
lucinexus config --curl "curl http://192.168.1.50:11434/v1/chat/completions \
  -d '{\"model\": \"llama3.2:3b\"}'"

# Example 3: Import Groq High-Speed Inference
lucinexus config --curl "curl https://api.groq.com/openai/v1/chat/completions \
  -H 'Authorization: Bearer gsk_...' \
  -d '{\"model\": \"llama-3.3-70b-versatile\"}'"

```

**Console Output:**

```
✓ Successfully parsed cURL command:
  Endpoint : https://api.deepseek.com/chat/completions
  Model    : deepseek-chat
  Auth     : Authorization: Bearer sk-****...
✓ Performing endpoint handshake test...
✓ Handshake successful! Latency: 142ms
✓ Configuration saved to /etc/config/lucinexus
✓ LuciNexus daemon reloaded. Ready for commands.

```

### 4.2 Method B: Granular Flag-Based Configuration

If you prefer setting parameters individually:

```
lucinexus config set \
  --endpoint="https://openrouter.ai/api/v1/chat/completions" \
  --model="anthropic/claude-3.5-sonnet" \
  --key="sk-or-v1-..." \
  --auth-header="Authorization" \
  --auth-prefix="Bearer "

```

### 4.3 Method C: Interactive Terminal Setup Wizard

Run `lucinexus setup` to enter a guided wizard:

```
lucinexus setup

```

**Interactive Walkthrough:**

```
=====================================================
  LuciNexus AI Provider Configuration Wizard
=====================================================
Choose setup method:
  1) Paste a cURL command (Recommended)
  2) Select predefined cloud provider (Gemini, OpenAI, Anthropic)
  3) Manually enter Custom / Local API details
Selection [1-3]: 1

Paste your cURL command and press ENTER:
> curl https://api.together.xyz/v1/chat/completions -H "Authorization: Bearer tog_..." -d '{"model": "meta-llama/Llama-3.3-70B-Instruct-Turbo"}'

Parsing cURL configuration...
  ✓ Endpoint: https://api.together.xyz/v1/chat/completions
  ✓ Model: meta-llama/Llama-3.3-70B-Instruct-Turbo
  ✓ Token: tog_****
Testing connection to endpoint... (Success: 98ms)
Apply configuration? [Y/n]: Y
Configuration active!

```

## 5. Modern LuCI Web Interface Integration

The LuCI web UI (`package/luci-app-lucinexus`) includes an interactive **"Custom AI & cURL Import"** tab.

### 5.1 LuCI Controller Endpoint (`luasrc/controller/lucinexus.lua`)

```
entry({"admin", "services", "lucinexus", "import_curl"}, call("action_import_curl")).leaf = true

function action_import_curl()
    local http = require "luci.http"
    local sys = require "luci.sys"
    local json = require "luci.jsonc"

    local raw_payload = http.content()
    local req = json.parse(raw_payload)

    if not req or not req.curl_cmd then
        http.status(400, "Bad Request")
        http.write('{"error":"Missing curl_cmd parameter"}')
        return
    end

    -- Invoke lucinexus internal parser utility
    local cmd = string.format('/usr/bin/lucinexus config --curl %s --json 2>&1', sys.shellquote(req.curl_cmd))
    local result = sys.exec(cmd)

    http.prepare_content("application/json")
    http.write(result)
end

```

### 5.2 LuCI Client-Side JavaScript Component (`htdocs/.../view/lucinexus/app.js`)

```
// cURL Parser Card Component in LuCI:
E('div', { 'class': 'cbi-section' }, [
    E('h3', {}, _('Custom AI Provider / cURL Auto-Import')),
    E('div', { 'class': 'cbi-section-descr' }, _('Paste a cURL command from your AI provider (e.g. DeepSeek, Groq, Ollama, OpenRouter) to automatically extract and apply configuration.')),
    
    E('div', { 'class': 'cbi-value' }, [
        E('label', { 'class': 'cbi-value-title' }, _('cURL Snippet')),
        E('div', { 'class': 'cbi-value-field' }, [
            E('textarea', {
                'id': 'curl_input',
                'class': 'cbi-input-textarea',
                'rows': 5,
                'style': 'width: 100%; font-family: monospace; font-size: 0.9rem;',
                'placeholder': 'curl https://api.deepseek.com/chat/completions \\\n  -H "Authorization: Bearer sk-..." \\\n  -d \'{"model": "deepseek-chat"}\''
            }),
            E('button', {
                'class': 'cbi-button cbi-button-action',
                'style': 'margin-top: 8px;',
                'click': function() {
                    var snippet = document.getElementById('curl_input').value;
                    if (!snippet) {
                        ui.addNotification(null, E('p', {}, _('Please paste a cURL command.')));
                        return;
                    }
                    ui.showIndicator('Parsing & Verifying Endpoint...');
                    fs.exec_direct('/usr/bin/lucinexus', ['config', '--curl', snippet])
                        .then(function(out) {
                            ui.hideIndicator();
                            ui.addNotification(null, E('p', {}, _('Configuration successfully imported!')), 'info');
                            location.reload();
                        })
                        .catch(function(err) {
                            ui.hideIndicator();
                            ui.addNotification(null, E('p', {}, _('Failed: ') + err.message), 'error');
                        });
                }
            }, _('Auto-Detect, Test & Apply'))
        ])
    ])
])

```

## 6. Real-World Provider Presets

Below are tested cURL snippets that can be pasted directly into LuciNexus:

### DeepSeek (Cloud)

```
lucinexus config --curl "curl https://api.deepseek.com/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer sk-your-deepseek-key' \
  -d '{\"model\": \"deepseek-chat\"}'"

```

### Groq (Ultra-Low Latency Cloud)

```
lucinexus config --curl "curl https://api.groq.com/openai/v1/chat/completions \
  -H 'Authorization: Bearer gsk_your-groq-key' \
  -d '{\"model\": \"llama-3.3-70b-versatile\"}'"

```

### Local Ollama (Self-Hosted on Home LAN)

```
lucinexus config --curl "curl http://192.168.1.100:11434/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{\"model\": \"qwen2.5-coder:7b\"}'"

```

### OpenRouter (Multi-Model Gateway)

```
lucinexus config --curl "curl https://openrouter.ai/api/v1/chat/completions \
  -H 'Authorization: Bearer sk-or-v1-...' \
  -H 'HTTP-Referer: https://openwrt.org' \
  -H 'X-Title: LuciNexus-Router' \
  -d '{\"model\": \"meta-llama/llama-3.3-70b-instruct\"}'"

```

### Cloudflare AI Gateway

```
lucinexus config --curl "curl https://gateway.ai.cloudflare.com/v1/{account_id}/{gateway_id}/openai/chat/completions \
  -H 'Authorization: Bearer {token}' \
  -d '{\"model\": \"gpt-4o-mini\"}'"

```