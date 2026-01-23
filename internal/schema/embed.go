package schema

import (
	"embed"
)

// Schemas are copied from schema/ submodule by `make generate`
//
//go:embed billing.schema.json provider.schema.json
var FS embed.FS

const (
	BillingSchemaFile  = "billing.schema.json"
	ProviderSchemaFile = "provider.schema.json"
)

func BillingSchema() ([]byte, error) {
	return FS.ReadFile(BillingSchemaFile)
}

func ProviderSchema() ([]byte, error) {
	return FS.ReadFile(ProviderSchemaFile)
}
