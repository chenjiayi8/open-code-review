package main

import (
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

func TestLoadOrCreateConfigPreservesLegacyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"provider":"old","llm":{"auth_token":"secret"},"language":"en"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadOrCreateConfig(path)
	if err != nil {
		t.Fatalf("loadOrCreateConfig: %v", err)
	}
	if cfg.Provider != "old" || cfg.Llm.AuthToken != "secret" || cfg.Language != "en" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
