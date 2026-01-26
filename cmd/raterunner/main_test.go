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
		ExitErrHandler: func(c *cli.Context, err error) {
			// Write error to stdout to capture it in tests (matches main.go behavior)
			if err != nil {
				outBuf.WriteString("Error: " + err.Error() + "\n")
			}
		},
	}

	fullArgs := append([]string{"raterunner"}, args...)
	err := app.Run(fullArgs)

	exitCode = 0
	if err != nil {
		exitCode = 1
		// Capture errors returned from app.Run() that weren't handled by ExitErrHandler
		if !strings.Contains(outBuf.String(), "Error:") {
			outBuf.WriteString("Error: " + err.Error() + "\n")
		}
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

// --- Apply command tests ---

func TestApply_NoArguments(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "--env", "sandbox", "--dry-run")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "missing required argument")
}

func TestApply_MissingEnvFlag(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "testdata/valid/billing_full.yaml", "--dry-run")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "env")
}

func TestApply_MissingDryRunFlag(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "testdata/valid/billing_full.yaml", "--env", "sandbox")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "dry-run")
}

func TestApply_InvalidEnv(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "--env", "staging", "--dry-run", "testdata/valid/billing_full.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "invalid environment")
}

func TestApply_UnsupportedProvider(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "--env", "sandbox", "--dry-run", "testdata/apply/billing_paddle.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "not supported yet")
	assertContains(t, stdout, "raterunner@akorchak.software")
}

func TestApply_NoProviders(t *testing.T) {
	stdout, _, exitCode := runApp("apply", "--env", "sandbox", "--dry-run", "testdata/valid/billing_optional_field.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "no providers specified")
}

func TestApply_MissingAPIKey(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv("STRIPE_SANDBOX_KEY")

	stdout, _, exitCode := runApp("apply", "--env", "sandbox", "--dry-run", "testdata/valid/billing_full.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "STRIPE_SANDBOX_KEY")
}

func TestApply_WrongKeyPrefix(t *testing.T) {
	// Set a production key for sandbox environment
	os.Setenv("STRIPE_SANDBOX_KEY", "sk_live_wrongprefix")
	defer os.Unsetenv("STRIPE_SANDBOX_KEY")

	stdout, _, exitCode := runApp("apply", "--env", "sandbox", "--dry-run", "testdata/valid/billing_full.yaml")

	assertExitCode(t, 1, exitCode)
	assertContains(t, stdout, "sandbox environment requires a test key")
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
		"testdata/apply/billing_paddle.yaml",
	}

	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("testdata file missing: %s", f)
		}
	}
}
