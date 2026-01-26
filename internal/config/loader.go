package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadBillingFile loads and parses a billing configuration file
func LoadBillingFile(filePath string) (*BillingConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("unsupported file extension: %s (use .yaml or .yml)", ext)
	}

	var config BillingConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}
