package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	defaultAIProvider      = "github_models"
	defaultGitHubModel     = "gpt-4o"
	defaultOpenRouterModel = "google/gemini-flash-1.5"
	defaultLMStudioURL     = "http://localhost:1234/v1"
	defaultEmailLanguage   = "pl"
)

// Config is application configuration loaded from YAML.
type Config struct {
	NotesDir         string `mapstructure:"notes_dir" yaml:"notes_dir"`
	AIProvider       string `mapstructure:"ai_provider" yaml:"ai_provider"`
	GitHubModel      string `mapstructure:"github_model" yaml:"github_model"`
	OpenRouterModel  string `mapstructure:"openrouter_model" yaml:"openrouter_model"`
	OpenRouterAPIKey string `mapstructure:"openrouter_api_key" yaml:"openrouter_api_key"`
	LMStudioURL      string `mapstructure:"lm_studio_url" yaml:"lm_studio_url"`
	LMStudioModel    string `mapstructure:"lm_studio_model" yaml:"lm_studio_model"`
	EmailLanguage    string `mapstructure:"email_language" yaml:"email_language"`
}

// DefaultConfigPath returns the platform-appropriate config file path.
func DefaultConfigPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return filepath.Join(userConfigDir, "recap", "config.yaml"), nil
}

// Load loads config from configPath (or the default path when empty).
// If the config file doesn't exist, it creates one with defaults.
func Load(configPath string) (*Config, error) {
	resolvedPath := configPath
	if resolvedPath == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return nil, err
		}
		resolvedPath = defaultPath
	}

	if _, err := os.Stat(resolvedPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config file: %w", err)
		}

		cfg, err := defaultConfig()
		if err != nil {
			return nil, err
		}

		if err := Save(cfg, resolvedPath); err != nil {
			return nil, err
		}

		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(resolvedPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := applyDefaultsAndNormalize(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes cfg as YAML to configPath (or default path when empty).
func Save(cfg *Config, configPath string) error {
	if cfg == nil {
		return errors.New("config cannot be nil")
	}

	resolvedPath := configPath
	if resolvedPath == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		resolvedPath = defaultPath
	}

	copyCfg := *cfg
	if err := applyDefaultsAndNormalize(&copyCfg); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(copyCfg)
	if err != nil {
		return fmt.Errorf("marshal config yaml: %w", err)
	}

	if err := os.WriteFile(resolvedPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// MigrateOldConfig migrates legacy config from notes to recap path.
func MigrateOldConfig() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return migrateOldConfigWithDir(userConfigDir)
}

func migrateOldConfigWithDir(configDir string) (string, error) {
	oldConfigPath := filepath.Join(configDir, "notes", "config.yaml")
	newConfigPath := filepath.Join(configDir, "recap", "config.yaml")

	if _, err := os.Stat(oldConfigPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("stat old config file: %w", err)
	}

	if _, err := os.Stat(newConfigPath); err == nil {
		return "", nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat new config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(newConfigPath), 0o755); err != nil {
		return "", fmt.Errorf("create new config directory: %w", err)
	}

	oldConfigData, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return "", fmt.Errorf("read old config file: %w", err)
	}

	if err := os.WriteFile(newConfigPath, oldConfigData, 0o600); err != nil {
		return "", fmt.Errorf("write new config file: %w", err)
	}

	return "Migrated config from ~/.config/notes/ to ~/.config/recap/", nil
}

func defaultConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir: %w", err)
	}

	return &Config{
		NotesDir:         filepath.Join(homeDir, "recap"),
		AIProvider:       defaultAIProvider,
		GitHubModel:      defaultGitHubModel,
		OpenRouterModel:  defaultOpenRouterModel,
		OpenRouterAPIKey: "",
		LMStudioURL:      defaultLMStudioURL,
		LMStudioModel:    "",
		EmailLanguage:    defaultEmailLanguage,
	}, nil
}

func applyDefaultsAndNormalize(cfg *Config) error {
	if cfg.AIProvider == "" {
		cfg.AIProvider = defaultAIProvider
	}

	if cfg.AIProvider != "github_models" && cfg.AIProvider != "openrouter" && cfg.AIProvider != "lm_studio" {
		cfg.AIProvider = defaultAIProvider
	}

	if cfg.GitHubModel == "" {
		cfg.GitHubModel = defaultGitHubModel
	}

	if cfg.OpenRouterModel == "" {
		cfg.OpenRouterModel = defaultOpenRouterModel
	}

	if cfg.LMStudioURL == "" {
		cfg.LMStudioURL = defaultLMStudioURL
	}

	if cfg.EmailLanguage == "" {
		cfg.EmailLanguage = defaultEmailLanguage
	}

	if cfg.EmailLanguage != "en" && cfg.EmailLanguage != "pl" && cfg.EmailLanguage != "no" {
		cfg.EmailLanguage = defaultEmailLanguage
	}

	if cfg.NotesDir == "" {
		cfg.NotesDir = "~/recap"
	}

	expandedNotesDir, err := expandHomePath(cfg.NotesDir)
	if err != nil {
		return err
	}
	cfg.NotesDir = expandedNotesDir

	return nil
}

func expandHomePath(value string) (string, error) {
	if value != "~" && !strings.HasPrefix(value, "~/") && !strings.HasPrefix(value, "~\\") {
		return value, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	if value == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, value[2:]), nil
}
