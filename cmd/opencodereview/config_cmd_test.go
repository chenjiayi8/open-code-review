package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetConfigValueLanguage(t *testing.T) {
	cfg := &Config{}
	if err := setConfigValue(cfg, "language", "zh-CN"); err != nil {
		t.Fatalf("setConfigValue: %v", err)
	}
	if cfg.Language != "zh-CN" {
		t.Fatalf("Language = %q", cfg.Language)
	}
}

func TestSetConfigValueTelemetry(t *testing.T) {
	cfg := &Config{}
	if err := setConfigValue(cfg, "telemetry.enabled", "true"); err != nil {
		t.Fatalf("set telemetry.enabled: %v", err)
	}
	if cfg.Telemetry == nil || !cfg.Telemetry.Enabled {
		t.Fatalf("Telemetry = %+v", cfg.Telemetry)
	}
}

func TestSetConfigValueRejectsDirectLLMAndProviderKeys(t *testing.T) {
	for _, key := range []string{"provider", "model", "providers.openai.api_key", "custom_providers.gateway.url", "llm.url", "llm.auth_token"} {
		t.Run(key, func(t *testing.T) {
			err := setConfigValue(&Config{}, key, "value")
			if err == nil || !strings.Contains(err.Error(), "unknown config key") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestRunConfigSetLanguageWritesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := runConfigSet("language", "en-US"); err != nil {
		t.Fatalf("runConfigSet: %v", err)
	}
	cfg, err := LoadAppConfig(filepath.Join(home, ".opencodereview", "config.json"))
	if err != nil {
		t.Fatalf("LoadAppConfig: %v", err)
	}
	if cfg.Language != "en-US" {
		t.Fatalf("Language = %q", cfg.Language)
	}
}

func TestRunConfigRejectsProviderSubcommand(t *testing.T) {
	err := runConfig([]string{"provider"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadAppConfigMissingReturnsNil(t *testing.T) {
	cfg, err := LoadAppConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || cfg != nil {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestLoadOrCreateConfigDropsLegacyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"provider":"old","llm":{"auth_token":"secret"},"language":"en"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		t.Fatalf("loadOrCreateConfig: %v", err)
	}
	if cfg.Language != "en" {
		t.Fatalf("Language = %q", cfg.Language)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "provider") || strings.Contains(string(data), "llm") || strings.Contains(string(data), "secret") {
		t.Fatalf("legacy fields leaked after load: %s", data)
	}
}

func TestRunConfigSetScrubsLegacySecretConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".opencodereview", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{
		"provider":"anthropic",
		"model":"claude",
		"providers":{"anthropic":{"api_key":"secret-provider"}},
		"custom_providers":{"gateway":{"url":"https://example.invalid","auth_header":"Bearer secret-custom"}},
		"llm":{"url":"https://llm.invalid","auth_token":"secret-llm"},
		"mcp_servers":{"github":{"command":"secret-cmd","env":["TOKEN=secret-mcp"]}},
		"language":"en"
	}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runConfigSet("telemetry.enabled", "true"); err != nil {
		t.Fatalf("runConfigSet: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"provider", "providers", "custom_providers", "llm", "mcp_servers", "secret-provider", "secret-custom", "secret-llm", "secret-mcp"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("config still contains %q after allowed mutation:\n%s", forbidden, data)
		}
	}
	if !strings.Contains(string(data), `"language": "en"`) || !strings.Contains(string(data), `"enabled": true`) {
		t.Fatalf("allowed language/telemetry settings not preserved:\n%s", data)
	}
}

func TestLoadAppConfigIgnoresLegacySecretConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"provider":"anthropic","llm":{"auth_token":"secret"},"mcp_servers":{"x":{"env":["TOKEN=secret"]}},"language":"en","telemetry":{"enabled":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadAppConfig(path)
	if err != nil {
		t.Fatalf("LoadAppConfig: %v", err)
	}
	if cfg == nil || cfg.Language != "en" || cfg.Telemetry == nil || !cfg.Telemetry.Enabled {
		t.Fatalf("cfg = %+v", cfg)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"provider", "llm", "mcp_servers", "secret"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("app config exposed legacy field %q: %s", forbidden, data)
		}
	}
}
