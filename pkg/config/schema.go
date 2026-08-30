package config

import (
	_ "embed"
)

const (
	SchemaFile     = "pastiche:file"
	SchemaService  = "pastiche:service"
	SchemaServer   = "pastiche:server"
	SchemaResource = "pastiche:resource"
	SchemaEndpoint = "pastiche:endpoint"
)

//go:embed pastiche.schema.json
var schemaData []byte

// Schema retrieves the contents of the JSON schema (draft 2020-12)
// for Pastiche configuration files
func Schema() []byte {
	return schemaData
}
