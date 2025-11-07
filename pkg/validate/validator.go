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
	var schemaErrors []Diagnostic

	// Validate for type errors and constraint violations
	if err := unified.Validate(); err != nil {
		schemaErrors = convertCueErrors(err, sourceName)
	}

	// Check for missing required fields (incomplete values)
	// CUE's Validate() doesn't catch missing required fields by default,
	// so we need to explicitly check for incomplete/concrete errors
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		// Only add errors that aren't already captured by the first Validate()
		// Check if this is a different set of errors
		incompleteErrors := convertCueErrors(err, sourceName)
		// Add incomplete errors that aren't duplicates
		existingMsgs := make(map[string]bool)
		for _, diag := range schemaErrors {
			existingMsgs[diag.Message] = true
		}
		for _, diag := range incompleteErrors {
			if !existingMsgs[diag.Message] {
				schemaErrors = append(schemaErrors, diag)
			}
		}
	}

	// Check for deprecated fields and add warnings
	deprecationWarnings := checkDeprecatedFields(yamlData, sourceName, data)

	// Combine schema errors and deprecation warnings
	allDiagnostics := append(schemaErrors, deprecationWarnings...)

	return allDiagnostics, nil
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

// checkDeprecatedFields checks for deprecated fields and returns warnings
func checkDeprecatedFields(yamlData interface{}, sourceName string, originalYAML []byte) []Diagnostic {
	var warnings []Diagnostic

	// Parse YAML with line information to get accurate line numbers
	var yamlNode yaml.Node
	if err := yaml.Unmarshal(originalYAML, &yamlNode); err != nil {
		// If we can't parse with line info, skip line numbers
		return checkDeprecatedFieldsRecursive(yamlData, sourceName, "")
	}

	// Check for deprecated fields
	if yamlNode.Kind == yaml.DocumentNode && len(yamlNode.Content) > 0 {
		root := yamlNode.Content[0]
		if root.Kind == yaml.MappingNode {
			for i := 0; i < len(root.Content); i += 2 {
				if i+1 >= len(root.Content) {
					break
				}
				keyNode := root.Content[i]
				valueNode := root.Content[i+1]
				if keyNode.Value == "runners" && valueNode.Kind == yaml.MappingNode {
					// Found runners map, check each runner for deprecated disk field
					for j := 0; j < len(valueNode.Content); j += 2 {
						if j+1 >= len(valueNode.Content) {
							break
						}
						_ = valueNode.Content[j] // runner key node (not used, but needed for iteration)
						runnerValueNode := valueNode.Content[j+1]
						if runnerValueNode.Kind == yaml.MappingNode {
							// Check if this runner has a disk field
							for k := 0; k < len(runnerValueNode.Content); k += 2 {
								if k >= len(runnerValueNode.Content) {
									break
								}
								fieldKeyNode := runnerValueNode.Content[k]
								if fieldKeyNode.Value == "disk" {
									// Found deprecated disk field
									warnings = append(warnings, Diagnostic{
										Path:     sourceName,
										Line:     fieldKeyNode.Line,
										Column:   fieldKeyNode.Column,
										Message:  "field 'disk' is deprecated, use 'volume' instead (e.g., volume=80gb:gp3:125mbs:3000iops)",
										Severity: SeverityWarning,
									})
								}
							}
						}
					}
				} else if keyNode.Value == "pools" && valueNode.Kind == yaml.MappingNode {
					// Found pools map, check each pool for deprecated environment field
					for j := 0; j < len(valueNode.Content); j += 2 {
						if j+1 >= len(valueNode.Content) {
							break
						}
						_ = valueNode.Content[j] // pool key node (not used, but needed for iteration)
						poolValueNode := valueNode.Content[j+1]
						if poolValueNode.Kind == yaml.MappingNode {
							// Check if this pool has an environment field
							for k := 0; k < len(poolValueNode.Content); k += 2 {
								if k >= len(poolValueNode.Content) {
									break
								}
								fieldKeyNode := poolValueNode.Content[k]
								if fieldKeyNode.Value == "environment" {
									// Found deprecated environment field
									warnings = append(warnings, Diagnostic{
										Path:     sourceName,
										Line:     fieldKeyNode.Line,
										Column:   fieldKeyNode.Column,
										Message:  "field 'environment' is deprecated, use 'env' instead",
										Severity: SeverityWarning,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return warnings
}

// checkDeprecatedFieldsRecursive is a fallback that checks without line numbers
func checkDeprecatedFieldsRecursive(data interface{}, sourceName string, path string) []Diagnostic {
	var warnings []Diagnostic

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			currentPath := path
			if currentPath != "" {
				currentPath += "."
			}
			currentPath += key

			if key == "runners" {
				// Check runners map
				if runnersMap, ok := value.(map[string]interface{}); ok {
					for runnerKey, runnerValue := range runnersMap {
						if runnerSpec, ok := runnerValue.(map[string]interface{}); ok {
							if _, hasDisk := runnerSpec["disk"]; hasDisk {
								warnings = append(warnings, Diagnostic{
									Path:     sourceName,
									Line:     0,
									Column:   0,
									Message:  fmt.Sprintf("field 'runners.%s.disk' is deprecated, use 'volume' instead (e.g., volume=80gb:gp3:125mbs:3000iops)", runnerKey),
									Severity: SeverityWarning,
								})
							}
						}
					}
				}
			} else if key == "pools" {
				// Check pools map
				if poolsMap, ok := value.(map[string]interface{}); ok {
					for poolKey, poolValue := range poolsMap {
						if poolSpec, ok := poolValue.(map[string]interface{}); ok {
							if _, hasEnvironment := poolSpec["environment"]; hasEnvironment {
								warnings = append(warnings, Diagnostic{
									Path:     sourceName,
									Line:     0,
									Column:   0,
									Message:  fmt.Sprintf("field 'pools.%s.environment' is deprecated, use 'env' instead", poolKey),
									Severity: SeverityWarning,
								})
							}
						}
					}
				}
			} else {
				// Recurse into nested structures
				warnings = append(warnings, checkDeprecatedFieldsRecursive(value, sourceName, currentPath)...)
			}
		}
	case map[interface{}]interface{}:
		for key, value := range v {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			currentPath := path
			if currentPath != "" {
				currentPath += "."
			}
			currentPath += keyStr

			if keyStr == "runners" {
				// Check runners map
				if runnersMap, ok := value.(map[interface{}]interface{}); ok {
					for runnerKey, runnerValue := range runnersMap {
						runnerKeyStr, ok := runnerKey.(string)
						if !ok {
							continue
						}
						if runnerSpec, ok := runnerValue.(map[interface{}]interface{}); ok {
							if _, hasDisk := runnerSpec["disk"]; hasDisk {
								warnings = append(warnings, Diagnostic{
									Path:     sourceName,
									Line:     0,
									Column:   0,
									Message:  fmt.Sprintf("field 'runners.%s.disk' is deprecated, use 'volume' instead (e.g., volume=80gb:gp3:125mbs:3000iops)", runnerKeyStr),
									Severity: SeverityWarning,
								})
							}
						}
					}
				}
			} else if keyStr == "pools" {
				// Check pools map
				if poolsMap, ok := value.(map[interface{}]interface{}); ok {
					for poolKey, poolValue := range poolsMap {
						poolKeyStr, ok := poolKey.(string)
						if !ok {
							continue
						}
						if poolSpec, ok := poolValue.(map[interface{}]interface{}); ok {
							if _, hasEnvironment := poolSpec["environment"]; hasEnvironment {
								warnings = append(warnings, Diagnostic{
									Path:     sourceName,
									Line:     0,
									Column:   0,
									Message:  fmt.Sprintf("field 'pools.%s.environment' is deprecated, use 'env' instead", poolKeyStr),
									Severity: SeverityWarning,
								})
							}
						}
					}
				}
			} else {
				// Recurse into nested structures
				warnings = append(warnings, checkDeprecatedFieldsRecursive(value, sourceName, currentPath)...)
			}
		}
	case []interface{}:
		for i, item := range v {
			warnings = append(warnings, checkDeprecatedFieldsRecursive(item, sourceName, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}

	return warnings
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
