package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProviderConfig represents the provider ID mapping file
type ProviderConfig struct {
	Provider    string               `yaml:"provider"`
	Environment string               `yaml:"environment"`
	SyncedAt    string               `yaml:"synced_at,omitempty"`
	Plans       map[string]PlanIDs   `yaml:"plans,omitempty"`
	Addons      map[string]ProductIDs `yaml:"addons,omitempty"`
	Promotions  map[string]string    `yaml:"promotions,omitempty"`
}

// PlanIDs contains Stripe IDs for a plan
type PlanIDs struct {
	ProductID string            `yaml:"product_id"`
	Prices    map[string]string `yaml:"prices,omitempty"` // interval -> price_id
}

// ProductIDs contains Stripe IDs for an addon
type ProductIDs struct {
	ProductID string `yaml:"product_id"`
	PriceID   string `yaml:"price_id"`
}

// ProviderDir returns the raterunner/ directory path for a billing config
func ProviderDir(billingPath string) string {
	dir := filepath.Dir(billingPath)
	return filepath.Join(dir, "raterunner")
}

// ProviderFilePath returns the provider file path for a billing config
func ProviderFilePath(billingPath, provider, env string) string {
	dir := filepath.Dir(billingPath)
	// If billingPath is inside raterunner/, use that directory
	if filepath.Base(dir) == "raterunner" {
		return filepath.Join(dir, fmt.Sprintf("%s_%s.yaml", provider, env))
	}
	// Otherwise, use raterunner/ subdirectory
	return filepath.Join(dir, "raterunner", fmt.Sprintf("%s_%s.yaml", provider, env))
}

// LoadProviderFile loads a provider config from a file
func LoadProviderFile(path string) (*ProviderConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("unsupported file extension: %s (use .yaml or .yml)", ext)
	}

	var cfg ProviderConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}

// SaveProviderFile saves a provider config to a file
func SaveProviderFile(path string, cfg *ProviderConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	content, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
