package validator

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests use embedded schemas via New()
// For schema override testing, use NewWithSchemaDir()

func createTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

func TestValidateBillingFile_Valid(t *testing.T) {
	content := `version: 1
entitlements:
  projects:
    type: int
    unit: project
  api_requests:
    type: rate
    unit: request
plans:
  - id: free
    name: Free Plan
    prices:
      monthly: { amount: 0 }
    limits:
      projects: 5
      api_requests: { limit: 100, per: minute }
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateBillingFile_MissingRequiredField(t *testing.T) {
	content := `version: 1
plans:
  - id: free
    prices:
      monthly: { amount: 0 }
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result")
	}

	found := false
	for _, e := range result.Errors {
		if containsStr(e.Message, "name") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing 'name', got: %v", result.Errors)
	}
}

func TestValidateBillingFile_InvalidVersion(t *testing.T) {
	content := `version: 99
plans:
  - id: free
    name: Free
    prices:
      monthly: { amount: 0 }
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for wrong version")
	}

	found := false
	for _, e := range result.Errors {
		if e.Path == "/version" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /version, got: %v", result.Errors)
	}
}

func TestValidateBillingFile_InvalidPlanID(t *testing.T) {
	content := `version: 1
plans:
  - id: INVALID_ID
    name: Test
    prices:
      monthly: { amount: 0 }
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for bad plan ID pattern")
	}

	found := false
	for _, e := range result.Errors {
		if e.Path == "/plans/0/id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error at /plans/0/id, got: %v", result.Errors)
	}
}

func TestValidateBillingFile_UndefinedEntitlement(t *testing.T) {
	content := `version: 1
entitlements:
  projects:
    type: int
plans:
  - id: pro
    name: Pro Plan
    prices:
      monthly: { amount: 1900 }
    limits:
      projects: 10
      undefined_limit: 50
      another_missing: true
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for undefined entitlements")
	}

	undefinedCount := 0
	for _, e := range result.Errors {
		if containsStr(e.Message, "undefined entitlement") {
			undefinedCount++
		}
	}
	if undefinedCount != 2 {
		t.Errorf("expected 2 undefined entitlement errors, got %d: %v", undefinedCount, result.Errors)
	}
}

func TestValidateBillingFile_UndefinedEntitlementInAddon(t *testing.T) {
	content := `version: 1
entitlements:
  projects:
    type: int
plans:
  - id: free
    name: Free
    prices:
      monthly: { amount: 0 }
addons:
  - id: extra_stuff
    name: Extra Stuff
    price: { amount: 500 }
    grants:
      projects: "+10"
      nonexistent: 5
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for undefined entitlement in addon")
	}

	found := false
	for _, e := range result.Errors {
		if containsStr(e.Path, "addons") && containsStr(e.Message, "nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about undefined 'nonexistent' in addon, got: %v", result.Errors)
	}
}

func TestValidateBillingFile_NoEntitlementsSection(t *testing.T) {
	content := `version: 1
plans:
  - id: free
    name: Free
    prices:
      monthly: { amount: 0 }
`
	file := createTempFile(t, "billing.yaml", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid when no entitlements defined, got errors: %v", result.Errors)
	}
}

func TestValidateProviderFile_Valid(t *testing.T) {
	content := `provider: stripe
environment: production
plans:
  pro:
    product_id: prod_123
`
	file := createTempFile(t, "stripe.prod.yaml", content)

	v := New()
	result, err := v.ValidateProviderFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateProviderFile_InvalidProvider(t *testing.T) {
	content := `provider: unknown_provider
environment: production
`
	file := createTempFile(t, "provider.yaml", content)

	v := New()
	result, err := v.ValidateProviderFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for unknown provider")
	}
}

func TestValidateFile_JSON(t *testing.T) {
	content := `{
  "version": 1,
  "plans": [
    {
      "id": "free",
      "name": "Free",
      "prices": {
        "monthly": { "amount": 0 }
      }
    }
  ]
}`
	file := createTempFile(t, "billing.json", content)

	v := New()
	result, err := v.ValidateBillingFile(file)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid JSON file, got errors: %v", result.Errors)
	}
}

func TestValidateFile_InvalidYAML(t *testing.T) {
	content := `version: 1
plans:
  - id: free
    name: "unclosed string
`
	file := createTempFile(t, "invalid.yaml", content)

	v := New()
	_, err := v.ValidateBillingFile(file)

	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidateFile_UnsupportedExtension(t *testing.T) {
	file := createTempFile(t, "config.txt", "some content")

	v := New()
	_, err := v.ValidateBillingFile(file)

	if err == nil {
		t.Error("expected error for unsupported file extension")
	}
}

func TestValidateFile_NonExistent(t *testing.T) {
	v := New()
	_, err := v.ValidateBillingFile("/nonexistent/file.yaml")

	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestValidationError_String(t *testing.T) {
	tests := []struct {
		err      ValidationError
		expected string
	}{
		{
			err:      ValidationError{Path: "/plans/0/id", Message: "invalid pattern"},
			expected: "/plans/0/id: invalid pattern",
		},
		{
			err:      ValidationError{Path: "/plans/0", Message: "undefined entitlement", Detail: "extra context"},
			expected: "/plans/0: undefined entitlement (extra context)",
		},
	}

	for _, tt := range tests {
		got := tt.err.String()
		if got != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, got)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
