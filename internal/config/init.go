package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitTemplate is the example billing.yaml content
const InitTemplate = `# Raterunner Billing Configuration
# Documentation: https://raterunner.run/docs
version: 1
providers:
  - stripe

# Define entitlements (features and limits)
# Types: int (numeric limit), bool (feature flag), rate (rate limit)
entitlements:
  projects:
    type: int
    unit: projects
    description: Number of projects allowed
  api_calls:
    type: int
    unit: requests
    description: Monthly API calls
  support:
    type: bool
    description: Access to priority support

# Define pricing plans
plans:
  - id: free
    name: Free
    description: Get started with basic features
    headline: Perfect for side projects
    type: personal
    public: true
    default: true
    prices:
      monthly:
        amount: 0
      yearly:
        amount: 0
    limits:
      projects: 3
      api_calls: 1000
      support: false
    features:
      - Up to 3 projects
      - 1,000 API calls/month
      - Community support

  - id: pro
    name: Pro
    description: For professionals and growing teams
    headline: Everything you need to scale
    type: personal
    public: true
    trial_days: 14
    prices:
      monthly:
        amount: 1900
      yearly:
        amount: 19000
    limits:
      projects: 50
      api_calls: 100000
      support: true
    features:
      - Up to 50 projects
      - 100,000 API calls/month
      - Email support
      - Priority features
    upgrades_to:
      - enterprise

  - id: enterprise
    name: Enterprise
    description: Custom solutions for large organizations
    headline: Tailored for your business
    type: enterprise
    public: false
    prices:
      monthly:
        amount: 0
      yearly:
        amount: 0
    limits:
      projects: unlimited
      api_calls: unlimited
      support: true
    features:
      - Unlimited projects
      - Unlimited API calls
      - Dedicated support
      - Custom integrations
      - SLA guarantee

  # Example: One-time payment (lifetime deal)
  # - id: lifetime
  #   name: Lifetime Deal
  #   description: Pay once, use forever
  #   headline: Limited time offer
  #   type: personal
  #   billing_model: one_time
  #   public: true
  #   prices:
  #     one_time:
  #       amount: 29900
  #   limits:
  #     projects: 50
  #     api_calls: 100000
  #     support: true
  #   features:
  #     - Lifetime access
  #     - All future updates
`

// CreateInitFiles creates the raterunner/billing.yaml file in the specified directory
func CreateInitFiles(dir string) error {
	// Create raterunner directory
	raterunnerDir := filepath.Join(dir, "raterunner")
	if err := os.MkdirAll(raterunnerDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write billing.yaml
	billingPath := filepath.Join(raterunnerDir, "billing.yaml")
	if err := os.WriteFile(billingPath, []byte(InitTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write billing.yaml: %w", err)
	}

	return nil
}

// InitFilesExist checks if raterunner/billing.yaml already exists
func InitFilesExist(dir string) bool {
	billingPath := filepath.Join(dir, "raterunner", "billing.yaml")
	_, err := os.Stat(billingPath)
	return err == nil
}

// InitFilePath returns the path to the billing.yaml that would be created
func InitFilePath(dir string) string {
	return filepath.Join(dir, "raterunner", "billing.yaml")
}
