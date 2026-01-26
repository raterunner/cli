package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func runApp(args ...string) (stdout, stderr string, exitCode int) {
	var outBuf, errBuf bytes.Buffer

	app := &cli.App{
		Name:      "raterunner",
		Usage:     "Raterunner CLI - billing configuration management",
		Version:   "0.1.0",
		Writer:    &outBuf,
		ErrWriter: &errBuf,
		Commands: []*cli.Command{
			{
				Name:      "validate",
				Usage:     "Validate a billing or provider configuration file",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "schema-dir",
						Aliases: []string{"s"},
						Usage:   "Path to directory containing schema files",
					},
				},
				Action: validateAction,
			},
		},
		ExitErrHandler: func(c *cli.Context, err error) {},
	}

	fullArgs := append([]string{"raterunner"}, args...)
	err := app.Run(fullArgs)

	exitCode = 0
	if err != nil {
		exitCode = 1
	}

	return outBuf.String(), errBuf.String(), exitCode
}

// --- Valid files ---

func TestValidate_ValidBillingMinimal(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/valid/billing_minimal.yaml")

	assertExitCode(t, 0, exitCode)
	assertContains(t, stdout, "is valid")
}

func TestValidate_ValidBillingFull(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/valid/billing_full.yaml")

	assertExitCode(t, 0, exitCode)
	assertContains(t, stdout, "is valid")
}

func TestValidate_ValidBillingJSON(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/valid/billing_minimal.json")

	assertExitCode(t, 0, exitCode)
	assertContains(t, stdout, "is valid")
}

func TestValidate_ValidProviderStripe(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/valid/provider_stripe.yaml")

	assertExitCode(t, 0, exitCode)
	assertContains(t, stdout, "is valid")
}

func TestValidate_ValidBillingOptionalField(t *testing.T) {
	// providers field is optional - this file has no providers section
	stdout, _, exitCode := runApp("validate", "testdata/valid/billing_optional_field.yaml")

	assertExitCode(t, 0, exitCode)
	assertContains(t, stdout, "is valid")
}

// --- Invalid files: schema errors ---

func TestValidate_MissingRequiredField(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_missing_name.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "validation error")
	assertContains(t, stdout, "name")
}

func TestValidate_InvalidVersion(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_bad_version.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "/version")
}

func TestValidate_InvalidPlanID(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_invalid_plan_id.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "/plans/0/id")
}

func TestValidate_InvalidProviderFile(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/provider_unknown.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "validation error")
}

func TestValidate_UnsupportedProviderInBilling(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_unsupported_provider.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "validation error")
}

// --- Invalid files: semantic errors ---

func TestValidate_UndefinedEntitlement(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_undefined_entitlement.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "undefined entitlement")
	assertContains(t, stdout, "unknown_feature")
}

func TestValidate_UndefinedEntitlementInAddon(t *testing.T) {
	stdout, _, exitCode := runApp("validate", "testdata/invalid/billing_undefined_entitlement_addon.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "undefined entitlement")
	assertContains(t, stdout, "nonexistent")
}

// --- File handling errors ---

func TestValidate_MalformedYAML(t *testing.T) {
	_, _, exitCode := runApp("validate", "testdata/errors/malformed.yaml")

	assertExitCode(t, 1, exitCode)
}

func TestValidate_NonExistentFile(t *testing.T) {
	_, _, exitCode := runApp("validate", "testdata/nonexistent.yaml")

	assertExitCode(t, 1, exitCode)
}

// --- CLI behavior ---

func TestValidate_NoArguments(t *testing.T) {
	_, _, exitCode := runApp("validate")

	assertExitCode(t, 1, exitCode)
}

// --- Test helpers ---

func assertExitCode(t *testing.T, expected, actual int) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected exit code %d, got %d", expected, actual)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got: %s", substr, s)
	}
}

// --- Verify testdata files exist ---

func TestTestdataFilesExist(t *testing.T) {
	files := []string{
		"testdata/valid/billing_minimal.yaml",
		"testdata/valid/billing_full.yaml",
		"testdata/valid/billing_minimal.json",
		"testdata/valid/billing_optional_field.yaml",
		"testdata/valid/provider_stripe.yaml",
		"testdata/invalid/billing_missing_name.yaml",
		"testdata/invalid/billing_bad_version.yaml",
		"testdata/invalid/billing_invalid_plan_id.yaml",
		"testdata/invalid/billing_undefined_entitlement.yaml",
		"testdata/invalid/billing_undefined_entitlement_addon.yaml",
		"testdata/invalid/billing_unsupported_provider.yaml",
		"testdata/invalid/provider_unknown.yaml",
		"testdata/errors/malformed.yaml",
	}

	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("testdata file missing: %s", f)
		}
	}
}
