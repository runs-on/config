package validate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/runs-on/config/pkg/validate"
)

func TestValidateFile_Valid(t *testing.T) {
	testFiles := []string{
		"../../schema/testdata/valid/basic.yml",
		"../../schema/testdata/valid/with-anchors.yml",
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
	testFile := "../../schema/testdata/invalid/basic.yml"
	diags, err := validate.ValidateFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ValidateFile failed: %v", err)
	}

	if len(diags) == 0 {
		t.Error("Expected diagnostics for invalid file, got none")
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

