package schemajson

import (
	_ "embed"
)

//go:embed ../../schema/schema.json
var schemaJSON []byte

// Schema returns the JSON schema as bytes
func Schema() []byte {
	return schemaJSON
}

