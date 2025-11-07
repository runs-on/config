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
		"../../schema/testdata/valid/github-private-runs-on.yml",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			diags, err := validate.ValidateFile(context.Background(), testFile)
			if err != nil {
				t.Fatalf("ValidateFile failed: %v", err)
			}

			// Filter out warnings - only check for errors
			errors := filterErrors(diags)
			if len(errors) > 0 {
				t.Errorf("Expected no errors for valid file, got %d:", len(errors))
				for _, diag := range errors {
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

	// Name is now optional, so pools without explicit name fields should be valid
	// However, we still need a runner reference, so check for that error instead
	if len(diags) > 0 {
		// If there are diagnostics, they should be about missing runner, not missing name
		for _, diag := range diags {
			if contains(diag.Message, "name") && contains(diag.Message, "required") {
				t.Errorf("Unexpected error about missing name (name is now optional): %v", diag)
			}
		}
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
	defer func() {
		//nolint:errcheck // Close errors in tests are safe to ignore
		_ = file.Close()
	}()

	diags, err := validate.ValidateReader(context.Background(), file, testFile)
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid file, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateFile_AllTopLevelFields(t *testing.T) {
	testFile := "../../schema/testdata/valid/all-top-level-fields.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for file with all top-level fields, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateFile_TopLevelFieldsIndividually(t *testing.T) {
	testFiles := []struct {
		name     string
		filePath string
	}{
		{"extends-only", "../../schema/testdata/valid/extends-only.yml"},
		{"runners-only", "../../schema/testdata/valid/runners-only.yml"},
		{"images-only", "../../schema/testdata/valid/images-only.yml"},
		{"pools-only", "../../schema/testdata/valid/pools-only.yml"},
		{"admins-only", "../../schema/testdata/valid/admins-only.yml"},
	}

	for _, tt := range testFiles {
		t.Run(tt.name, func(t *testing.T) {
			diags, err := validate.ValidateFile(context.Background(), tt.filePath)
			if err != nil {
				t.Fatalf("ValidateFile failed: %v", err)
			}

			errors := filterErrors(diags)
			if len(errors) > 0 {
				t.Errorf("Expected no errors for %s, got %d:", tt.name, len(errors))
				for _, diag := range errors {
					t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
				}
			}
		})
	}
}

func TestValidateFile_CustomFieldsAllowed(t *testing.T) {
	testFile := "../../schema/testdata/valid/with-custom-fields.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for file with custom fields (x-defaults, etc.), got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateReader_CustomFieldsAllowed(t *testing.T) {
	// Test with inline YAML that includes custom fields
	yamlContent := `x-defaults: &defaults
  cpu: [2]
  ram: [16]
  family: [c7a]

custom-field: "some value"
another-custom:
  nested: value

runners:
  test-runner:
    <<: *defaults
    image: ubuntu22-full-x64

images:
  test-image:
    ami: ami-1234567890abcdef0

pools:
  test-pool:
    name: test-pool
    runner: test-runner
    schedule:
      - name: default
        hot: 1
        stopped: 2

admins:
  - admin1
`

	reader := strings.NewReader(yamlContent)
	diags, err := validate.ValidateReader(context.Background(), reader, "test.yml")
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for YAML with custom fields and anchors, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateReader_AllTopLevelFields(t *testing.T) {
	yamlContent := `_extends: ".github-private"

runners:
  test-runner:
    cpu: [2]
    ram: [16]
    family: [c7a]

images:
  test-image:
    ami: ami-1234567890abcdef0

pools:
  test-pool:
    name: test-pool
    runner: test-runner
    schedule:
      - name: default
        hot: 1
        stopped: 2

admins:
  - admin1
  - admin2
`

	reader := strings.NewReader(yamlContent)
	diags, err := validate.ValidateReader(context.Background(), reader, "test.yml")
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for YAML with all top-level fields, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateReader_EachTopLevelField(t *testing.T) {
	testCases := []struct {
		name        string
		yamlContent string
	}{
		{
			name:        "_extends",
			yamlContent: `_extends: ".github-private"`,
		},
		{
			name: "runners",
			yamlContent: `runners:
  test-runner:
    cpu: [2]
    ram: [16]
    family: [c7a]`,
		},
		{
			name: "images",
			yamlContent: `images:
  test-image:
    ami: ami-1234567890abcdef0`,
		},
		{
			name: "pools",
			yamlContent: `runners:
  test-runner:
    cpu: [2]
    ram: [16]
    family: [c7a]
pools:
  test-pool:
    name: test-pool
    runner: test-runner
    schedule:
      - name: default
        hot: 1
        stopped: 2`,
		},
		{
			name: "admins",
			yamlContent: `admins:
  - admin1
  - admin2`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.yamlContent)
			diags, err := validate.ValidateReader(context.Background(), reader, "test.yml")
			if err != nil {
				t.Fatalf("ValidateReader failed: %v", err)
			}

			errors := filterErrors(diags)
			if len(errors) > 0 {
				t.Errorf("Expected no errors for %s field, got %d:", tc.name, len(errors))
				for _, diag := range errors {
					t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
				}
			}
		})
	}
}

// filterErrors returns only error-level diagnostics, filtering out warnings
func filterErrors(diags []validate.Diagnostic) []validate.Diagnostic {
	var errors []validate.Diagnostic
	for _, diag := range diags {
		if diag.Severity == validate.SeverityError {
			errors = append(errors, diag)
		}
	}
	return errors
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

