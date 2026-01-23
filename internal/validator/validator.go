package validator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"raterunner/internal/schema"
)

type ValidationError struct {
	Path    string
	Message string
	Detail  string
}

func (e ValidationError) String() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Path, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

type Validator struct {
	schemaFS  fs.FS
	schemaDir string // optional: override with filesystem path
}

// New creates a validator using embedded schemas
func New() *Validator {
	return &Validator{schemaFS: schema.FS}
}

// NewWithSchemaDir creates a validator using schemas from a filesystem directory
func NewWithSchemaDir(schemaDir string) *Validator {
	return &Validator{schemaDir: schemaDir}
}

func (v *Validator) ValidateBillingFile(filePath string) (*ValidationResult, error) {
	return v.validateFile(filePath, schema.BillingSchemaFile)
}

func (v *Validator) ValidateProviderFile(filePath string) (*ValidationResult, error) {
	return v.validateFile(filePath, schema.ProviderSchemaFile)
}

func (v *Validator) validateFile(filePath, schemaName string) (*ValidationResult, error) {
	data, err := loadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load file: %w", err)
	}

	result := &ValidationResult{Valid: true}

	schemaErrors, err := v.validateSchema(data, schemaName)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	if len(schemaErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, schemaErrors...)
	}

	if schemaName == schema.BillingSchemaFile {
		semanticErrors := validateBillingSemantics(data)
		if len(semanticErrors) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, semanticErrors...)
		}
	}

	return result, nil
}

func (v *Validator) loadSchema(schemaName string) ([]byte, error) {
	if v.schemaDir != "" {
		return os.ReadFile(filepath.Join(v.schemaDir, schemaName))
	}
	return fs.ReadFile(v.schemaFS, schemaName)
}

func loadFile(filePath string) (any, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var data any

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &data); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		data = convertYAMLToJSON(data)
	case ".json":
		if err := json.Unmarshal(content, &data); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file extension: %s (use .yaml, .yml, or .json)", ext)
	}

	return data, nil
}

func convertYAMLToJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = convertYAMLToJSON(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertYAMLToJSON(v)
		}
		return result
	default:
		return v
	}
}

func (v *Validator) validateSchema(data any, schemaName string) ([]ValidationError, error) {
	schemaContent, err := v.loadSchema(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	var schemaDoc any
	if err := json.Unmarshal(schemaContent, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaName, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema: %w", err)
	}

	sch, err := compiler.Compile(schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	err = sch.Validate(data)
	if err == nil {
		return nil, nil
	}

	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return nil, err
	}

	return extractSchemaErrors(validationErr), nil
}

func extractSchemaErrors(err *jsonschema.ValidationError) []ValidationError {
	var errors []ValidationError

	var extract func(e *jsonschema.ValidationError)
	extract = func(e *jsonschema.ValidationError) {
		path := "/" + strings.Join(e.InstanceLocation, "/")
		if path == "/" {
			path = "(root)"
		}

		if len(e.Causes) == 0 && e.ErrorKind != nil {
			msg := formatErrorKind(e.ErrorKind)
			errors = append(errors, ValidationError{
				Path:    path,
				Message: msg,
			})
		}

		for _, cause := range e.Causes {
			extract(cause)
		}
	}

	extract(err)
	return errors
}

func formatErrorKind(kind jsonschema.ErrorKind) string {
	s := fmt.Sprintf("%+v", kind)
	s = strings.TrimPrefix(s, "&")
	s = strings.ReplaceAll(s, "kind.", "")

	switch {
	case strings.HasPrefix(s, "{Missing:"):
		fields := strings.TrimPrefix(s, "{Missing:[")
		fields = strings.TrimSuffix(fields, "]}")
		return fmt.Sprintf("missing required field(s): %s", fields)
	case strings.HasPrefix(s, "{Got:") && strings.Contains(s, "Want:"):
		s = strings.TrimPrefix(s, "{")
		s = strings.TrimSuffix(s, "}")
		parts := strings.Split(s, " Want:")
		if len(parts) == 2 {
			got := strings.TrimPrefix(parts[0], "Got:")
			want := parts[1]
			if strings.HasPrefix(want, "[") {
				return fmt.Sprintf("got '%s', expected one of: %s", got, want)
			}
			return fmt.Sprintf("got '%s', expected '%s'", got, want)
		}
	}
	return s
}

func validateBillingSemantics(data any) []ValidationError {
	var errors []ValidationError

	root, ok := data.(map[string]any)
	if !ok {
		return errors
	}

	definedEntitlements := make(map[string]bool)
	if entitlements, ok := root["entitlements"].(map[string]any); ok {
		for key := range entitlements {
			definedEntitlements[key] = true
		}
	}

	if len(definedEntitlements) == 0 {
		return errors
	}

	if plans, ok := root["plans"].([]any); ok {
		for i, plan := range plans {
			planMap, ok := plan.(map[string]any)
			if !ok {
				continue
			}
			planID := "unknown"
			if id, ok := planMap["id"].(string); ok {
				planID = id
			}

			if limits, ok := planMap["limits"].(map[string]any); ok {
				for key := range limits {
					if !definedEntitlements[key] {
						errors = append(errors, ValidationError{
							Path:    fmt.Sprintf("/plans/%d/limits/%s", i, key),
							Message: fmt.Sprintf("undefined entitlement '%s'", key),
							Detail:  fmt.Sprintf("plan '%s' references entitlement '%s' which is not defined in the entitlements section", planID, key),
						})
					}
				}
			}
		}
	}

	if addons, ok := root["addons"].([]any); ok {
		for i, addon := range addons {
			addonMap, ok := addon.(map[string]any)
			if !ok {
				continue
			}
			addonID := "unknown"
			if id, ok := addonMap["id"].(string); ok {
				addonID = id
			}

			if grants, ok := addonMap["grants"].(map[string]any); ok {
				for key := range grants {
					if !definedEntitlements[key] {
						errors = append(errors, ValidationError{
							Path:    fmt.Sprintf("/addons/%d/grants/%s", i, key),
							Message: fmt.Sprintf("undefined entitlement '%s'", key),
							Detail:  fmt.Sprintf("addon '%s' grants entitlement '%s' which is not defined in the entitlements section", addonID, key),
						})
					}
				}
			}
		}
	}

	return errors
}
