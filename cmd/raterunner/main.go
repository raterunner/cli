package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"raterunner/internal/config"
	"raterunner/internal/diff"
	"raterunner/internal/stripe"
	"raterunner/internal/validator"
)

func main() {
	app := &cli.App{
		Name:    "raterunner",
		Usage:   "Raterunner CLI - billing configuration management",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:      "validate",
				Usage:     "Validate a billing or provider configuration file",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "schema-dir",
						Aliases: []string{"s"},
						Usage:   "Path to directory containing schema files (uses embedded schemas if not specified)",
					},
				},
				Action: validateAction,
			},
			{
				Name:      "apply",
				Usage:     "Compare local billing config with remote Stripe state",
				ArgsUsage: "<billing.yaml>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "env",
						Aliases:  []string{"e"},
						Usage:    "Environment: sandbox or production",
						Required: true,
					},
					&cli.BoolFlag{
						Name:     "dry-run",
						Usage:    "Preview changes without applying (required for now)",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output as JSON instead of table",
					},
				},
				Action: applyAction,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func validateAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing required argument: file path")
	}

	filePath := c.Args().First()
	schemaDir := c.String("schema-dir")
	schemaType := detectSchemaType(filePath)

	var v *validator.Validator
	if schemaDir != "" {
		v = validator.NewWithSchemaDir(schemaDir)
	} else {
		v = validator.New()
	}

	var result *validator.ValidationResult
	var err error

	switch schemaType {
	case "billing":
		result, err = v.ValidateBillingFile(filePath)
	case "provider":
		result, err = v.ValidateProviderFile(filePath)
	default:
		return fmt.Errorf("unknown schema type: %s (use 'billing' or 'provider')", schemaType)
	}

	if err != nil {
		return err
	}

	out := c.App.Writer
	if out == nil {
		out = os.Stdout
	}

	if result.Valid {
		fmt.Fprintf(out, "✓ %s is valid\n", filePath)
		return nil
	}

	fmt.Fprintf(out, "✗ %s has %d validation error(s):\n\n", filePath, len(result.Errors))
	for i, e := range result.Errors {
		fmt.Fprintf(out, "  %d. %s\n", i+1, e.String())
	}
	fmt.Fprintln(out)

	return cli.Exit("", 1)
}

func detectSchemaType(filePath string) string {
	filename := strings.ToLower(filepath.Base(filePath))
	// Detect provider config files by filename prefix
	providerPrefixes := []string{"provider_", "stripe_", "paddle_", "chargebee_"}
	for _, prefix := range providerPrefixes {
		if strings.HasPrefix(filename, prefix) {
			return "provider"
		}
	}
	return "billing"
}

func applyAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing required argument: billing config file path")
	}

	filePath := c.Args().First()
	env := c.String("env")
	jsonOutput := c.Bool("json")

	out := c.App.Writer
	if out == nil {
		out = os.Stdout
	}

	// Validate environment
	var stripeEnv stripe.Environment
	switch env {
	case "sandbox":
		stripeEnv = stripe.Sandbox
	case "production":
		stripeEnv = stripe.Production
	default:
		return fmt.Errorf("invalid environment: %s (use 'sandbox' or 'production')", env)
	}

	// Load billing config
	cfg, err := config.LoadBillingFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load billing config: %w", err)
	}

	// Validate provider
	if err := validateProvider(cfg.Providers); err != nil {
		return err
	}

	// Get API key from environment
	apiKey, err := getAPIKey(stripeEnv)
	if err != nil {
		return err
	}

	// Create Stripe client
	client, err := stripe.NewClient(stripeEnv, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Stripe client: %w", err)
	}

	// Fetch products with prices from Stripe
	products, err := client.FetchProductsWithPrices()
	if err != nil {
		return fmt.Errorf("failed to fetch from Stripe: %w", err)
	}

	// Compare
	result := diff.Compare(cfg, products, env)

	// Output
	if jsonOutput {
		if err := diff.OutputJSON(out, result); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
	} else {
		diff.OutputTable(out, result)
	}

	// Exit with code 1 if there are differences
	if result.HasDifferences() {
		return cli.Exit("", 1)
	}

	return nil
}

func validateProvider(providers []string) error {
	if len(providers) == 0 {
		return fmt.Errorf("no providers specified in billing config")
	}

	for _, p := range providers {
		switch p {
		case "stripe":
			// OK
		case "paddle", "chargebee":
			return fmt.Errorf("provider '%s' is not supported yet. Contact raterunner@akorchak.software if you need support", p)
		default:
			return fmt.Errorf("unknown provider: %s", p)
		}
	}

	// Check that stripe is in the list
	hasStripe := false
	for _, p := range providers {
		if p == "stripe" {
			hasStripe = true
			break
		}
	}
	if !hasStripe {
		return fmt.Errorf("billing config must include 'stripe' provider for apply command")
	}

	return nil
}

func getAPIKey(env stripe.Environment) (string, error) {
	var envVar string
	switch env {
	case stripe.Sandbox:
		envVar = "STRIPE_SANDBOX_KEY"
	case stripe.Production:
		envVar = "STRIPE_PRODUCTION_KEY"
	}

	key := os.Getenv(envVar)
	if key == "" {
		return "", fmt.Errorf("environment variable %s is not set", envVar)
	}

	return key, nil
}
