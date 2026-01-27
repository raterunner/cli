package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CLISettings represents persistent CLI configuration
type CLISettings struct {
	Quiet bool `yaml:"quiet,omitempty" json:"quiet,omitempty"`
}

// DefaultSettingsPath returns the default path for CLI settings
func DefaultSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".raterunner.yaml"
	}
	return filepath.Join(home, ".raterunner", "config.yaml")
}

// LoadSettings loads CLI settings from the default path
func LoadSettings() (*CLISettings, error) {
	return LoadSettingsFrom(DefaultSettingsPath())
}

// LoadSettingsFrom loads CLI settings from a specific path
func LoadSettingsFrom(path string) (*CLISettings, error) {
	settings := &CLISettings{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil // Return defaults if file doesn't exist
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// SaveSettings saves CLI settings to the default path
func SaveSettings(settings *CLISettings) error {
	return SaveSettingsTo(DefaultSettingsPath(), settings)
}

// SaveSettingsTo saves CLI settings to a specific path
func SaveSettingsTo(path string, settings *CLISettings) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
