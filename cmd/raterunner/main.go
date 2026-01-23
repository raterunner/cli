package main

import (
	"fmt"
	"os"
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
					&cli.StringFlag{
						Name:    "type",
						Aliases: []string{"t"},
						Usage:   "Schema type: billing or provider (auto-detected if not specified)",
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
	schemaType := c.String("type")

	if schemaType == "" {
		schemaType = detectSchemaType(filePath)
	}

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

	if result.Valid {
		fmt.Printf("✓ %s is valid\n", filePath)
		return nil
	}

	fmt.Printf("✗ %s has %d validation error(s):\n\n", filePath, len(result.Errors))
	for i, e := range result.Errors {
		fmt.Printf("  %d. %s\n", i+1, e.String())
	}
	fmt.Println()

	return cli.Exit("", 1)
}

func detectSchemaType(filePath string) string {
	lower := strings.ToLower(filePath)
	if strings.Contains(lower, "provider") || strings.Contains(lower, "stripe") ||
		strings.Contains(lower, "paddle") || strings.Contains(lower, "chargebee") {
		return "provider"
	}
	return "billing"
}
