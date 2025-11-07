package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/runs-on/config/pkg/validate"
)

func main() {
	var (
		format  = flag.String("format", "text", "Output format: text, json, or sarif")
		stdin   = flag.Bool("stdin", false, "Read from stdin instead of file")
		version = flag.Bool("version", false, "Print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Println("runs-on-config-lint v0.1.0")
		os.Exit(0)
	}

	var diags []validate.Diagnostic
	var err error
	ctx := context.Background()

	if *stdin {
		diags, err = validate.ValidateReader(ctx, os.Stdin, "<stdin>")
	} else {
		if flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "Error: no file specified\n")
			flag.Usage()
			os.Exit(1)
		}
		filePath := flag.Arg(0)
		diags, err = validate.ValidateFile(ctx, filePath)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Count errors (warnings don't cause failure)
	errorCount := 0
	for _, diag := range diags {
		if diag.Severity == validate.SeverityError {
			errorCount++
		}
	}

	exitCode := 0
	if errorCount > 0 {
		exitCode = 1
	}

	switch *format {
	case "text":
		outputText(diags)
	case "json":
		outputJSON(diags)
	case "sarif":
		outputSARIF(diags)
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid format %q (valid: text, json, sarif)\n", *format)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func outputText(diags []validate.Diagnostic) {
	if len(diags) == 0 {
		fmt.Println("✓ No issues found")
		return
	}

	// Separate errors and warnings
	var errors []validate.Diagnostic
	var warnings []validate.Diagnostic

	for _, diag := range diags {
		if diag.Severity == validate.SeverityError {
			errors = append(errors, diag)
		} else {
			warnings = append(warnings, diag)
		}
	}

	// Print errors first
	if len(errors) > 0 {
		fmt.Printf("\n✗ Found %d error(s):\n\n", len(errors))
		for i, diag := range errors {
			loc := formatLocation(diag)
			fmt.Printf("  %d. %s\n", i+1, loc)
			fmt.Printf("     %s\n", diag.Message)
			if i < len(errors)-1 {
				fmt.Println()
			}
		}
	}

	// Print warnings
	if len(warnings) > 0 {
		if len(errors) > 0 {
			fmt.Println()
		}
		fmt.Printf("⚠ Found %d warning(s):\n\n", len(warnings))
		for i, diag := range warnings {
			loc := formatLocation(diag)
			fmt.Printf("  %d. %s\n", i+1, loc)
			fmt.Printf("     %s\n", diag.Message)
			if i < len(warnings)-1 {
				fmt.Println()
			}
		}
	}

	// Print summary
	fmt.Println()
	if len(errors) > 0 {
		fmt.Printf("✗ Validation failed with %d error(s)", len(errors))
		if len(warnings) > 0 {
			fmt.Printf(" and %d warning(s)", len(warnings))
		}
		fmt.Println()
	} else {
		fmt.Printf("✓ Validation passed with %d warning(s)\n", len(warnings))
	}
}

func formatLocation(diag validate.Diagnostic) string {
	if diag.Line > 0 {
		return fmt.Sprintf("%s:%d:%d", diag.Path, diag.Line, diag.Column)
	}
	return diag.Path
}

func outputJSON(diags []validate.Diagnostic) {
	type jsonDiagnostic struct {
		Path     string `json:"path"`
		Line     int    `json:"line,omitempty"`
		Column   int    `json:"column,omitempty"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}

	type jsonOutput struct {
		Valid       bool             `json:"valid"`
		Diagnostics []jsonDiagnostic `json:"diagnostics"`
	}

	output := jsonOutput{
		Valid:       len(diags) == 0,
		Diagnostics: make([]jsonDiagnostic, len(diags)),
	}

	for i, diag := range diags {
		output.Diagnostics[i] = jsonDiagnostic{
			Path:     diag.Path,
			Line:     diag.Line,
			Column:   diag.Column,
			Message:  diag.Message,
			Severity: string(diag.Severity),
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputSARIF(diags []validate.Diagnostic) {
	// Basic SARIF output - can be enhanced later
	type sarifLocation struct {
		URI   string `json:"uri"`
		Region struct {
			StartLine   int `json:"startLine,omitempty"`
			StartColumn int `json:"startColumn,omitempty"`
		} `json:"region,omitempty"`
	}

	type sarifResult struct {
		RuleID    string        `json:"ruleId"`
		Level     string        `json:"level"`
		Message   struct {
			Text string `json:"text"`
		} `json:"message"`
		Locations []struct {
			PhysicalLocation sarifLocation `json:"physicalLocation"`
		} `json:"locations"`
	}

	type sarifRun struct {
		Tool struct {
			Driver struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"driver"`
		} `json:"tool"`
		Results []sarifResult `json:"results"`
	}

	type sarifOutput struct {
		Version string   `json:"version"`
		Runs    []sarifRun `json:"runs"`
	}

	results := make([]sarifResult, len(diags))
	for i, diag := range diags {
		level := "error"
		if diag.Severity == validate.SeverityWarning {
			level = "warning"
		}

		result := sarifResult{
			RuleID: "config-validation",
			Level:  level,
		}
		result.Message.Text = diag.Message

		loc := sarifLocation{
			URI: diag.Path,
		}
		if diag.Line > 0 {
			loc.Region.StartLine = diag.Line
			loc.Region.StartColumn = diag.Column
		}

		result.Locations = []struct {
			PhysicalLocation sarifLocation `json:"physicalLocation"`
		}{
			{PhysicalLocation: loc},
		}

		results[i] = result
	}

	output := sarifOutput{
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: struct {
					Driver struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					} `json:"driver"`
				}{
					Driver: struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					}{
						Name:    "runs-on-config-lint",
						Version: "0.1.0",
					},
				},
				Results: results,
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding SARIF: %v\n", err)
		os.Exit(1)
	}
}

