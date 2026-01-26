package stripe

import (
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v82"
)

// Environment represents the Stripe environment
type Environment string

const (
	Sandbox    Environment = "sandbox"
	Production Environment = "production"
)

// Client wraps the Stripe API client
type Client struct {
	env Environment
}

// NewClient creates a new Stripe client for the given environment
func NewClient(env Environment, apiKey string) (*Client, error) {
	if err := validateKey(env, apiKey); err != nil {
		return nil, err
	}

	stripe.Key = apiKey

	return &Client{env: env}, nil
}

// validateKey validates that the API key prefix matches the environment
func validateKey(env Environment, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is empty")
	}

	switch env {
	case Sandbox:
		if !strings.HasPrefix(apiKey, "sk_test_") {
			return fmt.Errorf("sandbox environment requires a test key (sk_test_...), got key with prefix '%s'", keyPrefix(apiKey))
		}
	case Production:
		if !strings.HasPrefix(apiKey, "sk_live_") {
			return fmt.Errorf("production environment requires a live key (sk_live_...), got key with prefix '%s'", keyPrefix(apiKey))
		}
	default:
		return fmt.Errorf("unknown environment: %s", env)
	}

	return nil
}

// keyPrefix extracts the prefix from an API key for error messages
func keyPrefix(key string) string {
	if len(key) > 8 {
		return key[:8] + "..."
	}
	return key + "..."
}

// GetEnv returns the environment this client is configured for
func (c *Client) GetEnv() Environment {
	return c.env
}
