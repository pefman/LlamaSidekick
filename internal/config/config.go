package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for LlamaSidekick
type Config struct {
	Ollama OllamaConfig `mapstructure:"ollama"`
	Models ModelsConfig `mapstructure:"models"`
	UI     UIConfig     `mapstructure:"ui"`
}

// OllamaConfig holds Ollama-specific settings
type OllamaConfig struct {
	Host        string  `mapstructure:"host"`
	Model       string  `mapstructure:"model"`        // Default model (deprecated, use Models config)
	Temperature float64 `mapstructure:"temperature"`
	Debug       bool    `mapstructure:"debug"`
}

// ModelsConfig holds per-mode model settings
type ModelsConfig struct {
	Plan  string `mapstructure:"plan"`
	Edit  string `mapstructure:"edit"`
	Agent string `mapstructure:"agent"`
	CMD   string `mapstructure:"cmd"`
}

// UIConfig holds UI-specific settings
type UIConfig struct {
	Theme string `mapstructure:"theme"`
}

// GetModelForMode returns the configured model for a specific mode
func (c *Config) GetModelForMode(mode string) string {
	switch mode {
	case "plan":
		if c.Models.Plan != "" {
			return c.Models.Plan
		}
	case "edit":
		if c.Models.Edit != "" {
			return c.Models.Edit
		}
	case "agent":
		if c.Models.Agent != "" {
			return c.Models.Agent
		}
	case "cmd":
		if c.Models.CMD != "" {
			return c.Models.CMD
		}
	}
	// Fallback to default model
	if c.Ollama.Model != "" {
		return c.Ollama.Model
	}
	return "codellama:7b"
}

// GetConfigDir returns the cross-platform config directory
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	
	llamaConfigDir := filepath.Join(configDir, "llamasidekick")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(llamaConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}
	
	return llamaConfigDir, nil
}

// GetDataDir returns the cross-platform data directory
func GetDataDir() (string, error) {
	// On Windows, UserConfigDir returns %APPDATA%, which we can use for data too
	// On Linux, we'll use ~/.local/share/llamasidekick
	var dataDir string
	
	if os.Getenv("XDG_DATA_HOME") != "" {
		dataDir = filepath.Join(os.Getenv("XDG_DATA_HOME"), "llamasidekick")
	} else if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".local", "share")); err == nil {
			dataDir = filepath.Join(home, ".local", "share", "llamasidekick")
		} else {
			// Fallback to config dir on Windows
			configDir, err := GetConfigDir()
			if err != nil {
				return "", err
			}
			dataDir = configDir
		}
	} else {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data dir: %w", err)
	}
	
	return dataDir, nil
}

// Load reads or creates the config file
func Load() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	
	configPath := filepath.Join(configDir, "config.yaml")
	isFirstRun := false
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		isFirstRun = true
	}
	
	// Set defaults
	viper.SetDefault("ollama.host", "http://localhost:11434")
	viper.SetDefault("ollama.model", "codellama:7b")
	viper.SetDefault("ollama.temperature", 0.7)
	viper.SetDefault("ollama.debug", false)
	viper.SetDefault("models.plan", "")
	viper.SetDefault("models.edit", "")
	viper.SetDefault("models.agent", "")
	viper.SetDefault("models.cmd", "")
	viper.SetDefault("ui.theme", "default")
	
	// Try to read config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Mark as first run for caller to handle model selection
	if isFirstRun {
		cfg.Ollama.Model = "" // Empty signals first run
	}
	
	return &cfg, nil
}

// Save saves the current config to disk
func (c *Config) Save() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	
	// Update viper with current values
	viper.Set("ollama.host", c.Ollama.Host)
	viper.Set("ollama.model", c.Ollama.Model)
	viper.Set("ollama.temperature", c.Ollama.Temperature)
	viper.Set("ollama.debug", c.Ollama.Debug)
	viper.Set("models.plan", c.Models.Plan)
	viper.Set("models.edit", c.Models.Edit)
	viper.Set("models.agent", c.Models.Agent)
	viper.Set("models.cmd", c.Models.CMD)
	viper.Set("ui.theme", c.UI.Theme)
	
	configPath := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	return nil
}
