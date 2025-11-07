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

