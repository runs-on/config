package validate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runs-on/config/pkg/validate"
)

func TestValidateFile_Valid(t *testing.T) {
	testFiles := []string{
		"../../schema/testdata/valid/basic.yml",
		"../../schema/testdata/valid/with-anchors.yml",
		"../../schema/testdata/valid/pool-complete.yml",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			diags, err := validate.ValidateFile(context.Background(), testFile)
			if err != nil {
				t.Fatalf("ValidateFile failed: %v", err)
			}

			if len(diags) > 0 {
				t.Errorf("Expected no diagnostics for valid file, got %d:", len(diags))
				for _, diag := range diags {
					t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
				}
			}
		})
	}
}

func TestValidateFile_Invalid(t *testing.T) {
	testFiles := []string{
		"../../schema/testdata/invalid/basic.yml",
		"../../schema/testdata/invalid/pool-missing-runner.yml",
		"../../schema/testdata/invalid/pool-missing-name.yml",
		"../../schema/testdata/invalid/pool-invalid-schedule.yml",
		"../../schema/testdata/invalid/pool-empty-schedule-name.yml",
		"../../schema/testdata/invalid/indentation-issue.yml",
		"../../schema/testdata/invalid/indentation-nested.yml",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			diags, err := validate.ValidateFile(context.Background(), testFile)
			if err != nil {
				t.Fatalf("ValidateFile failed: %v", err)
			}

			if len(diags) == 0 {
				t.Error("Expected diagnostics for invalid file, got none")
			} else {
				t.Logf("Found %d diagnostics for %s:", len(diags), testFile)
				for _, diag := range diags {
					t.Logf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
				}
			}
		})
	}
}

func TestValidateFile_PoolMissingRunner(t *testing.T) {
	testFile := "../../schema/testdata/invalid/pool-missing-runner.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	if len(diags) == 0 {
		t.Fatal("Expected diagnostics for pool missing runner, got none")
	}

	// Check that we get an error about missing runner
	foundRunnerError := false
	for _, diag := range diags {
		if contains(diag.Message, "runner") || contains(diag.Message, "required") {
			foundRunnerError = true
			break
		}
	}

	if !foundRunnerError {
		t.Errorf("Expected error about missing runner, got diagnostics: %v", diags)
	}
}

func TestValidateFile_PoolMissingName(t *testing.T) {
	testFile := "../../schema/testdata/invalid/pool-missing-name.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	if len(diags) == 0 {
		t.Fatal("Expected diagnostics for pool missing name, got none")
	}

	// Check that we get an error about missing name
	foundNameError := false
	for _, diag := range diags {
		if contains(diag.Message, "name") || contains(diag.Message, "required") {
			foundNameError = true
			break
		}
	}

	if !foundNameError {
		t.Errorf("Expected error about missing name, got diagnostics: %v", diags)
	}
}

func TestValidateFile_PoolInvalidSchedule(t *testing.T) {
	testFile := "../../schema/testdata/invalid/pool-invalid-schedule.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	if len(diags) == 0 {
		t.Fatal("Expected diagnostics for invalid schedule, got none")
	}

	// Check that we get errors about negative values
	foundNegativeError := false
	for _, diag := range diags {
		if contains(diag.Message, ">=0") || contains(diag.Message, "negative") || contains(diag.Message, "-5") || contains(diag.Message, "-10") {
			foundNegativeError = true
			break
		}
	}

	if !foundNegativeError {
		t.Errorf("Expected error about negative schedule values, got diagnostics: %v", diags)
	}
}

func TestValidateFile_IndentationIssues(t *testing.T) {
	testFiles := []string{
		"../../schema/testdata/invalid/indentation-issue.yml",
		"../../schema/testdata/invalid/indentation-nested.yml",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			diags, err := validate.ValidateFile(context.Background(), testFile)
			if err != nil {
				t.Fatalf("ValidateFile failed: %v", err)
			}

			// Indentation issues might cause YAML parse errors or schema validation errors
			// Either is acceptable - the important thing is we catch the problem
			if len(diags) == 0 {
				t.Error("Expected diagnostics for indentation issues, got none")
			}
		})
	}
}

func TestValidateReader(t *testing.T) {
	testFile := "../../schema/testdata/valid/basic.yml"
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	diags, err := validate.ValidateReader(context.Background(), file, testFile)
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	if len(diags) > 0 {
		t.Errorf("Expected no diagnostics for valid file, got %d:", len(diags))
		for _, diag := range diags {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

