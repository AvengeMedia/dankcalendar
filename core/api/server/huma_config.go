package server

import (
	"github.com/danielgtaylor/huma/v2"
)

func NewHumaConfig(title, version string) huma.Config {
	schemaPrefix := "#/components/schemas/"
	schemasPath := "/schemas"

	registry := huma.NewMapRegistry(schemaPrefix, huma.DefaultSchemaNamer)

	cfg := huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info: &huma.Info{
				Title:   title,
				Version: version,
			},
			Components: &huma.Components{
				Schemas: registry,
			},
		},
		OpenAPIPath:   "/openapi",
		DocsPath:      "/docs",
		SchemasPath:   schemasPath,
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
	}
	return cfg
}
