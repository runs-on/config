package validate

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/literal"
	"go.yaml.in/yaml/v3"
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

	// Parse YAML once: the node tree maps CUE errors back to source
	// positions, and decoding it expands anchors
	var doc yaml.Node
	var yamlData any
	err = yaml.Unmarshal(data, &doc)
	if err == nil {
		err = doc.Decode(&yamlData)
	}
	if err != nil {
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
		schemaErrors = convertCueErrors(err, sourceName, &doc)
	}

	// Check for missing required fields (incomplete values)
	// CUE's Validate() doesn't catch missing required fields by default,
	// so we need to explicitly check for incomplete/concrete errors
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		// Only add errors that aren't already captured by the first Validate()
		// Check if this is a different set of errors
		incompleteErrors := convertCueErrors(err, sourceName, &doc)
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
	deprecationWarnings := checkDeprecatedFields(&doc, sourceName)

	// Check for invalid runner references in pools
	runnerReferenceErrors := checkRunnerReferences(yamlData, sourceName)

	// Combine all diagnostics
	allDiagnostics := append(schemaErrors, deprecationWarnings...)
	allDiagnostics = append(allDiagnostics, runnerReferenceErrors...)

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

// convertCueErrors converts CUE validation errors to Diagnostic slice.
// The data was encoded from plain Go values, so CUE's own positions point into
// the schema; each error's path is mapped back to the YAML document instead.
func convertCueErrors(err error, sourceName string, doc *yaml.Node) []Diagnostic {
	var diagnostics []Diagnostic

	// CUE uses errors.List for multiple errors
	errList := errors.Errors(err)
	for _, err := range errList {
		line, column := yamlPosition(doc, errors.Path(err))

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

// yamlPosition returns the position of the YAML key or list item that a CUE
// error path names, such as ["#Config", "pools", "a", "name"]. A path ending
// in a field the file does not set, such as a missing required field, resolves
// to its deepest existing ancestor. It returns 0, 0 when nothing matches.
func yamlPosition(doc *yaml.Node, path []string) (line, column int) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return 0, 0
	}
	if len(path) > 0 && path[0] == "#Config" {
		path = path[1:]
	}

	node := doc.Content[0]
	for _, selector := range path {
		var located *yaml.Node
		switch node = resolveAlias(node); node.Kind {
		case yaml.MappingNode:
			// CUE quotes labels that are not identifiers, such as "small-x64".
			label := selector
			if unquoted, err := literal.Unquote(selector); err == nil {
				label = unquoted
			}
			located, node = mappingEntry(node, label)
		case yaml.SequenceNode:
			if i, err := strconv.Atoi(selector); err == nil && i >= 0 && i < len(node.Content) {
				located, node = node.Content[i], node.Content[i]
			}
		}
		if located == nil {
			break
		}
		line, column = located.Line, located.Column
	}
	return line, column
}

// mappingEntry returns the key and value nodes for key in a YAML mapping.
// Keys set through "<<" merge keys resolve to the anchored mapping that sets
// them, with explicit keys taking precedence as they do when decoding.
func mappingEntry(mapping *yaml.Node, key string) (keyNode, valueNode *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if k := mapping.Content[i]; k.ShortTag() != "!!merge" && resolveAlias(k).Value == key {
			return k, mapping.Content[i+1]
		}
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].ShortTag() != "!!merge" {
			continue
		}
		merged := resolveAlias(mapping.Content[i+1])
		sources := []*yaml.Node{merged}
		if merged.Kind == yaml.SequenceNode {
			sources = merged.Content
		}
		for _, source := range sources {
			if source = resolveAlias(source); source.Kind == yaml.MappingNode {
				if k, v := mappingEntry(source, key); k != nil {
					return k, v
				}
			}
		}
	}
	return nil, nil
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

// checkRunnerReferences checks that pool runners exist in the runners map
func checkRunnerReferences(yamlData any, sourceName string) []Diagnostic {
	var errors []Diagnostic

	// Type assert to map
	data, ok := yamlData.(map[string]any)
	if !ok {
		return errors
	}

	// Check if pools exist
	poolsRaw, hasPools := data["pools"]
	if !hasPools {
		return errors // No pools, nothing to validate
	}

	pools, ok := poolsRaw.(map[string]any)
	if !ok {
		return errors
	}

	// If pools exist, get the runners map
	runnersRaw, hasRunners := data["runners"]
	if !hasRunners {
		// If there are pools but no runners map, that's an error
		for poolName, poolValue := range pools {
			if pool, ok := poolValue.(map[string]any); ok {
				if runnerName, hasRunner := pool["runner"]; hasRunner {
					errors = append(errors, Diagnostic{
						Path:     sourceName,
						Line:     0,
						Column:   0,
						Message:  fmt.Sprintf("pool '%s' references runner '%v' but no runners are defined", poolName, runnerName),
						Severity: SeverityError,
					})
				}
			}
		}
		return errors
	}

	runners, ok := runnersRaw.(map[string]any)
	if !ok {
		return errors
	}

	// Check each pool's runner reference
	for poolName, poolValue := range pools {
		pool, ok := poolValue.(map[string]any)
		if !ok {
			continue
		}

		runnerName, hasRunner := pool["runner"]
		if !hasRunner {
			continue // Runner is required by schema, will be caught elsewhere
		}

		runnerNameStr, ok := runnerName.(string)
		if !ok {
			continue // Invalid type, will be caught by schema
		}

		// Check if the runner exists in the runners map
		if _, exists := runners[runnerNameStr]; !exists {
			errors = append(errors, Diagnostic{
				Path:     sourceName,
				Line:     0,
				Column:   0,
				Message:  fmt.Sprintf("pool '%s' references runner '%s' which is not defined in runners", poolName, runnerNameStr),
				Severity: SeverityError,
			})
		}
	}

	return errors
}

// checkDeprecatedFields checks for deprecated fields and returns warnings
func checkDeprecatedFields(doc *yaml.Node, sourceName string) []Diagnostic {
	var warnings []Diagnostic

	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root := doc.Content[0]
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
										Message:  "field 'disk' is deprecated and ignored; use 'volume' instead (e.g., volume=80gb:gp3:125mibps:3000iops)",
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

// normalizeSpotValues recursively normalizes boolean spot values to strings
// This allows YAML files to use spot: false (boolean) which gets converted to spot: "false" (string)
func normalizeSpotValues(data any) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			if key == "runners" {
				// Handle runners map specially to normalize spot values
				if runnersMap, ok := value.(map[string]any); ok {
					normalizedRunners := make(map[string]any)
					for runnerKey, runnerValue := range runnersMap {
						if runnerSpec, ok := runnerValue.(map[string]any); ok {
							normalizedSpec := make(map[string]any)
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
	case map[any]any:
		result := make(map[any]any)
		for key, value := range v {
			if keyStr, ok := key.(string); ok && keyStr == "runners" {
				// Handle runners map specially to normalize spot values
				if runnersMap, ok := value.(map[any]any); ok {
					normalizedRunners := make(map[any]any)
					for runnerKey, runnerValue := range runnersMap {
						if runnerSpec, ok := runnerValue.(map[any]any); ok {
							normalizedSpec := make(map[any]any)
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
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeSpotValues(item)
		}
		return result
	default:
		return v
	}
}
