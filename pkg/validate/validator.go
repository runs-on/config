package validate

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"gopkg.in/yaml.v3"
)

//go:embed schema.cue
var schemaFS embed.FS

// Diagnostic represents a validation error or warning
type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

// Severity indicates the severity of a diagnostic
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// ValidateFile validates a runs-on.yml file at the given path
func ValidateFile(ctx context.Context, filePath string) ([]Diagnostic, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return ValidateReader(ctx, file, filePath)
}

// ValidateReader validates YAML content from a reader
func ValidateReader(ctx context.Context, r io.Reader, sourceName string) ([]Diagnostic, error) {
	// Read the YAML content
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Parse YAML (this will expand anchors automatically)
	var yamlData interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return []Diagnostic{
			{
				Path:     sourceName,
				Line:     0,
				Column:   0,
				Message:  fmt.Sprintf("YAML parse error: %v", err),
				Severity: SeverityError,
			},
		}, nil
	}

	// Normalize boolean spot values to strings (CUE schema expects strings)
	yamlData = normalizeSpotValues(yamlData)

	// Re-marshal and unmarshal to ensure types are properly converted
	// This ensures boolean values are properly converted to strings
	normalizedYAML, err := yaml.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized YAML: %w", err)
	}
	if err := yaml.Unmarshal(normalizedYAML, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal normalized YAML: %w", err)
	}

	// Load CUE schema
	schema, err := loadSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// Create CUE context and compile the data
	ctx2 := cuecontext.New()
	dataValue := ctx2.Encode(yamlData)

	// Unify with schema and validate
	unified := schema.Unify(dataValue)
	if err := unified.Validate(); err != nil {
		return convertCueErrors(err, sourceName), nil
	}

	return []Diagnostic{}, nil
}

// loadSchema loads and compiles the CUE schema
func loadSchema() (cue.Value, error) {
	ctx := cuecontext.New()

	// Try to load embedded schema first
	var schemaData []byte
	var err error
	schemaData, err = schemaFS.ReadFile("schema.cue")
	if err != nil {
		// Fallback to file system (for development)
		paths := []string{"schema/runs_on.cue", "../../schema/runs_on.cue", "runs_on.cue"}
		for _, path := range paths {
			if data, err := os.ReadFile(path); err == nil {
				schemaData = data
				break
			}
		}
		if len(schemaData) == 0 {
			return cue.Value{}, fmt.Errorf("failed to read schema file")
		}
	}

	// Compile the schema
	value := ctx.CompileBytes(schemaData)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("failed to compile schema: %w", value.Err())
	}

	// Get the #Config definition
	config := value.LookupPath(cue.ParsePath("#Config"))
	if !config.Exists() {
		return cue.Value{}, fmt.Errorf("schema does not define #Config")
	}

	return config, nil
}

// convertCueErrors converts CUE validation errors to Diagnostic slice
func convertCueErrors(err error, sourceName string) []Diagnostic {
	var diagnostics []Diagnostic

	// CUE uses errors.List for multiple errors
	errList := errors.Errors(err)
	for _, err := range errList {
		pos := errors.Positions(err)
		line := 0
		column := 0
		if len(pos) > 0 {
			line = pos[0].Line()
			column = pos[0].Column()
		}

		msg := err.Error()
		// Clean up CUE error messages
		msg = strings.TrimPrefix(msg, "#Config:")
		msg = strings.TrimSpace(msg)

		diagnostics = append(diagnostics, Diagnostic{
			Path:     sourceName,
			Line:     line,
			Column:   column,
			Message:  msg,
			Severity: SeverityError,
		})
	}

	return diagnostics
}

// normalizeSpotValues recursively normalizes boolean spot values to strings
// This allows YAML files to use spot: false (boolean) which gets converted to spot: "false" (string)
func normalizeSpotValues(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if key == "runners" {
				// Handle runners map specially to normalize spot values
				if runnersMap, ok := value.(map[string]interface{}); ok {
					normalizedRunners := make(map[string]interface{})
					for runnerKey, runnerValue := range runnersMap {
						if runnerSpec, ok := runnerValue.(map[string]interface{}); ok {
							normalizedSpec := make(map[string]interface{})
							for specKey, specValue := range runnerSpec {
								if specKey == "spot" {
									// Convert boolean to string
									if spotBool, ok := specValue.(bool); ok {
										if spotBool {
											normalizedSpec[specKey] = "true"
										} else {
											normalizedSpec[specKey] = "false"
										}
									} else {
										normalizedSpec[specKey] = normalizeSpotValues(specValue)
									}
								} else {
									normalizedSpec[specKey] = normalizeSpotValues(specValue)
								}
							}
							normalizedRunners[runnerKey] = normalizedSpec
						} else {
							normalizedRunners[runnerKey] = normalizeSpotValues(runnerValue)
						}
					}
					result[key] = normalizedRunners
				} else {
					result[key] = normalizeSpotValues(value)
				}
			} else {
				result[key] = normalizeSpotValues(value)
			}
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for key, value := range v {
			if keyStr, ok := key.(string); ok && keyStr == "runners" {
				// Handle runners map specially to normalize spot values
				if runnersMap, ok := value.(map[interface{}]interface{}); ok {
					normalizedRunners := make(map[interface{}]interface{})
					for runnerKey, runnerValue := range runnersMap {
						if runnerSpec, ok := runnerValue.(map[interface{}]interface{}); ok {
							normalizedSpec := make(map[interface{}]interface{})
							for specKey, specValue := range runnerSpec {
								if specKeyStr, ok := specKey.(string); ok && specKeyStr == "spot" {
									// Convert boolean to string
									if spotBool, ok := specValue.(bool); ok {
										if spotBool {
											normalizedSpec[specKey] = "true"
										} else {
											normalizedSpec[specKey] = "false"
										}
									} else {
										normalizedSpec[specKey] = normalizeSpotValues(specValue)
									}
								} else {
									normalizedSpec[specKey] = normalizeSpotValues(specValue)
								}
							}
							normalizedRunners[runnerKey] = normalizedSpec
						} else {
							normalizedRunners[runnerKey] = normalizeSpotValues(runnerValue)
						}
					}
					result[key] = normalizedRunners
				} else {
					result[key] = normalizeSpotValues(value)
				}
			} else {
				result[key] = normalizeSpotValues(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeSpotValues(item)
		}
		return result
	default:
		return v
	}
}
