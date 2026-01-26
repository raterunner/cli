package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

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
