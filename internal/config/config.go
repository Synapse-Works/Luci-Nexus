package config

import (
	"encoding/json"
	"os"
	"strconv"
)

var (
	Version   = "1.0.0"
	BuildTime = "2026-11-01"
)

type ProviderType string

const (
	ProviderGemini    ProviderType = "gemini"
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderCustom    ProviderType = "custom"
	ProviderLocal     ProviderType = "local"
)

type Config struct {
	Provider        ProviderType      `json:"provider"`
	APIKey          string            `json:"api_key"`
	Model           string            `json:"model"`
	Endpoint        string            `json:"endpoint,omitempty"`
	AuthHeader      string            `json:"auth_header,omitempty"`
	AuthPrefix      string            `json:"auth_prefix,omitempty"`
	Protocol        string            `json:"protocol,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	TimeoutSec      int               `json:"timeout_sec"`
	MaxRetries      int               `json:"max_retries"`
	DryRunDefault   bool              `json:"dry_run_default"`
	AutoRollback    bool              `json:"auto_rollback"`
	EnableFastpath  bool              `json:"enable_fastpath"`
	SparseHydration bool              `json:"sparse_hydration"`
	ServerPort      int               `json:"server_port"`
	LogLevel        string            `json:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		Provider:        ProviderGemini,
		Model:           "gemini-2.5-flash",
		AuthHeader:      "Authorization",
		AuthPrefix:      "Bearer ",
		Protocol:        "openai-compatible",
		TimeoutSec:      25,
		MaxRetries:      2,
		DryRunDefault:   true,
		AutoRollback:    true,
		EnableFastpath:  true,
		SparseHydration: true,
		ServerPort:      9999,
		LogLevel:        "info",
		CustomHeaders:   make(map[string]string),
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			_ = json.Unmarshal(data, cfg)
		}
	}

	if key := os.Getenv("LUCINEXUS_API_KEY"); key != "" {
		cfg.APIKey = key
	} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		cfg.APIKey = key
		cfg.Provider = ProviderGemini
	} else if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		cfg.APIKey = key
		cfg.Provider = ProviderOpenAI
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.APIKey = key
		cfg.Provider = ProviderAnthropic
	}

	if p := os.Getenv("LUCINEXUS_PROVIDER"); p != "" {
		cfg.Provider = ProviderType(p)
	}
	if m := os.Getenv("LUCINEXUS_MODEL"); m != "" {
		cfg.Model = m
	}
	if ep := os.Getenv("LUCINEXUS_ENDPOINT"); ep != "" {
		cfg.Endpoint = ep
	}
	if dr := os.Getenv("LUCINEXUS_DRY_RUN"); dr != "" {
		cfg.DryRunDefault, _ = strconv.ParseBool(dr)
	}

	return cfg, nil
}
