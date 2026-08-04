package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long:  "Configuration management.\n\nOnly non-secret local application settings are supported:\n  ocr config set language en-US\n  ocr config set telemetry.enabled true",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:     "set <key> <value>",
	Short:   "Set a non-secret configuration value",
	Example: "  ocr config set language en-US\n  ocr config set telemetry.enabled true",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(args[0], args[1])
	},
}

var configUnsetCmd = &cobra.Command{
	Use:     "unset <key>",
	Short:   "Remove a non-secret configuration value",
	Example: "  ocr config unset language\n  ocr config unset telemetry.enabled",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigUnset(args[0])
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".opencodereview", "config.json"), nil
}

func resolveConfigPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("OCR_CONFIG_PATH")); p != "" {
		return p, nil
	}
	return defaultConfigPath()
}

func runConfigSet(key, value string) error {
	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}
	if err := saveConfig(configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigUnset(key string) error {
	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrCreateConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	switch key {
	case "language", "Language":
		cfg.Language = ""
	case "telemetry.enabled", "telemetry.Enabled":
		if cfg.Telemetry != nil {
			cfg.Telemetry.Enabled = false
		}
	case "telemetry.exporter", "telemetry.Exporter":
		if cfg.Telemetry != nil {
			cfg.Telemetry.Exporter = ""
		}
	case "telemetry.otlp_endpoint", "telemetry.OTLPEndpoint":
		if cfg.Telemetry != nil {
			cfg.Telemetry.OTLPEndpoint = ""
		}
	case "telemetry.content_logging", "telemetry.ContentLog":
		if cfg.Telemetry != nil {
			cfg.Telemetry.ContentLog = false
		}
	default:
		return unsupportedConfigKeyError(key)
	}
	if err := saveConfig(configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("Unset %s\n", key)
	return nil
}

type ProviderEntry struct {
	APIKey       string            `json:"api_key,omitempty"`
	URL          string            `json:"url,omitempty"`
	Protocol     string            `json:"protocol,omitempty"`
	Model        string            `json:"model,omitempty"`
	Models       []string          `json:"models,omitempty"`
	AuthHeader   string            `json:"auth_header,omitempty"`
	TimeoutSec   int               `json:"timeout_sec,omitempty"`
	ExtraBody    map[string]any    `json:"extra_body,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

type MCPServerConfig struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     []string          `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Tools   []string          `json:"tools,omitempty"`
	Setup   string            `json:"setup,omitempty"`
}

type Config struct {
	Provider        string                     `json:"provider,omitempty"`
	Model           string                     `json:"model,omitempty"`
	Providers       map[string]ProviderEntry   `json:"providers,omitempty"`
	CustomProviders map[string]ProviderEntry   `json:"custom_providers,omitempty"`
	Llm             LlmConfig                  `json:"llm,omitempty"`
	Language        string                     `json:"language,omitempty"`
	Telemetry       *TelemetryConfig           `json:"telemetry,omitempty"`
	MCPServers      map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
}

type LlmConfig struct {
	URL          string            `json:"url,omitempty"`
	AuthToken    string            `json:"auth_token,omitempty"`
	AuthHeader   string            `json:"auth_header,omitempty"`
	Model        string            `json:"model,omitempty"`
	Protocol     string            `json:"protocol,omitempty"`
	UseAnthropic *bool             `json:"use_anthropic,omitempty"`
	TimeoutSec   int               `json:"timeout_sec,omitempty"`
	ExtraBody    map[string]any    `json:"extra_body,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
}

type TelemetryConfig struct {
	Enabled      bool   `json:"enabled,omitempty"`
	Exporter     string `json:"exporter,omitempty"`
	OTLPEndpoint string `json:"otlp_endpoint,omitempty"`
	ContentLog   bool   `json:"content_logging,omitempty"`
}

func loadOrCreateConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func LoadAppConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read app config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse app config: %w", err)
	}
	return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

var supportedConfigKeys = []string{
	"language",
	"telemetry.enabled",
	"telemetry.exporter",
	"telemetry.otlp_endpoint",
	"telemetry.content_logging",
}

func setConfigValue(cfg *Config, key, value string) error {
	switch key {
	case "language", "Language":
		cfg.Language = value
	case "telemetry.enabled", "telemetry.Enabled":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for telemetry.enabled: %w", err)
		}
		cfg.ensureTelemetry()
		cfg.Telemetry.Enabled = b
	case "telemetry.exporter", "telemetry.Exporter":
		cfg.ensureTelemetry()
		cfg.Telemetry.Exporter = value
	case "telemetry.otlp_endpoint", "telemetry.OTLPEndpoint":
		cfg.ensureTelemetry()
		cfg.Telemetry.OTLPEndpoint = value
	case "telemetry.content_logging", "telemetry.ContentLog":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for telemetry.content_logging: %w", err)
		}
		cfg.ensureTelemetry()
		cfg.Telemetry.ContentLog = b
	default:
		return unsupportedConfigKeyError(key)
	}
	return nil
}

func unsupportedConfigKeyError(key string) error {
	return fmt.Errorf("unknown config key: %s\nSupported keys: %s", key, strings.Join(supportedConfigKeys, ", "))
}

func (c *Config) ensureTelemetry() {
	if c.Telemetry == nil {
		c.Telemetry = &TelemetryConfig{}
	}
}
