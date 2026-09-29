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
		"../../schema/testdata/valid/pool-runner-reference.yml",
		"../../schema/testdata/valid/nested-virt.yml",
		"../../schema/testdata/valid/spot-priorities.yml",
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
		"../../schema/testdata/invalid/pool-empty.yml",
		"../../schema/testdata/invalid/pool-non-ascii-name.yml",
		"../../schema/testdata/invalid/pool-invalid-runner-reference.yml",
		"../../schema/testdata/invalid/indentation-issue.yml",
		"../../schema/testdata/invalid/indentation-nested.yml",
		"../../schema/testdata/invalid/nested-virt.yml",
		"../../schema/testdata/invalid/spot-strategy.yml",
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

func TestValidateReader_RunnerAllFields(t *testing.T) {
	// Test all possible runner fields as documented in https://runs-on.com/configuration/job-labels/
	yamlContent := `runners:
  comprehensive-runner:
    # Basic resource fields
    cpu: [2, 4, 8]
    ram: [16, 32]
    family: [c7a, m7a]

    # Image and volume
    image: ubuntu22-full-x64
    volume: "80gb:gp3:125mibps:3000iops"
    sticky: "go-cache:20gb:gp3:750mibps:6000iops"

    # Deprecated disk field (should still validate but show ignored warning)
    disk: large

    # Retry configuration
    retry: ["always", "on-failure"]

    # Spot configuration
    spot: "price-capacity-optimized"

    # Network and access
    ssh: true
    private: false

    # Extras
    extras: ["s3-cache", "ecr-cache", "efs", "tmpfs"]

    # Debug mode
    debug: true

    # Additional fields
    preinstall: |
      apt-get update
      apt-get install -y docker
    prerun: |
      echo prepare-runner
      systemctl restart docker
    tags: ["Team:DevOps", "Environment:Production"]
    id: custom-runner-id

pools:
  test-pool:
    runner: comprehensive-runner
    schedule:
      - name: default
        hot: 1
        stopped: 2
`

	reader := strings.NewReader(yamlContent)
	diags, err := validate.ValidateReader(context.Background(), reader, "test.yml")
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for runner with all fields, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

func TestValidateReader_ImageAllFields(t *testing.T) {
	yamlContent := `images:
  comprehensive-image:
    platform: linux
    arch: x64
    name: ubuntu-22.04
    owner: "123456789012"
    preinstall: |
      apt-get update
      apt-get install -y docker
    prerun: |
      echo prepare-boot
      systemctl restart docker
    ami: ami-1234567890abcdef0
    tags:
      Team: DevOps
      Environment: Production
`

	reader := strings.NewReader(yamlContent)
	diags, err := validate.ValidateReader(context.Background(), reader, "test.yml")
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}

	errors := filterErrors(diags)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for image with all fields, got %d:", len(errors))
		for _, diag := range errors {
			t.Errorf("  %s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message)
		}
	}
}

// runnerYAML is the base document for per-field runner cases; imageYAML is the
// same for images. withFields nests field lines under the spec.
const (
	runnerYAML = "runners:\n  test:\n"
	imageYAML  = "images:\n  test:\n    ami: ami-1234567890abcdef0\n"
)

func withFields(base, fields string) string {
	return base + "    " + strings.ReplaceAll(fields, "\n", "\n    ") + "\n"
}

func TestValidateReader_RunnerFieldsIndividually(t *testing.T) {
	// cpu/ram have no numeric bounds and string fields have no format
	// constraints, so there are no out-of-range or bad-format rows for them.
	testCases := []struct {
		name    string
		fields  string
		wantErr string // substring of an expected error; empty means valid
	}{
		{"family", "family: c7a", ""},
		{"family-multiple", `family: ["c7a", "m7a"]`, ""},
		{"family-plus-separated", `family: "c7a+m7a"`, ""},
		{"family-int", "family: 7", "runners.test.family"},
		{"family-int-list", "family: [7]", "runners.test.family"},

		{"cpu", "cpu: 4", ""},
		{"cpu-array", "cpu: [2, 4, 8]", ""},
		{"cpu-plus-separated", `cpu: "2+4"`, ""},
		{"cpu-bool", "cpu: true", "runners.test.cpu"},
		{"cpu-bool-list", "cpu: [true]", "runners.test.cpu"},

		{"ram", "ram: 16", ""},
		{"ram-array", "ram: [16, 32]", ""},
		{"ram-plus-separated", `ram: "16+32"`, ""},
		{"ram-bool", "ram: true", "runners.test.ram"},
		{"ram-map", "ram: {gb: 16}", "runners.test.ram"},

		{"image", "image: ubuntu22-full-x64", ""},
		{"image-list", "image: [ubuntu22-full-x64]", "runners.test.image"},

		{"volume", `volume: "80gb:gp3:125mibps:3000iops"`, ""},
		{"volume-int", "volume: 80", "runners.test.volume"},

		{"retry", `retry: "always"`, ""},
		{"retry-array", `retry: ["always", "on-failure"]`, ""},
		{"retry-plus-separated", `retry: "always+on-failure"`, ""},
		{"retry-bool", "retry: false", ""},
		{"retry-int", "retry: 3", "runners.test.retry"},
		{"retry-int-list", "retry: [3]", "runners.test.retry"},

		{"spot-false", "spot: false", ""},
		{"spot-true", "spot: true", ""},
		{"spot-pco", `spot: "pco"`, ""},
		{"spot-price-capacity-optimized", `spot: "price-capacity-optimized"`, ""},
		{"spot-lowest-price", `spot: "lowest-price"`, ""},
		{"spot-lp", `spot: "lp"`, ""},
		{"spot-capacity-optimized", `spot: "capacity-optimized"`, ""},
		{"spot-co", `spot: "co"`, ""},
		{"spot-cop", `spot: "cop"`, ""},
		{"spot-capacity-optimized-prioritized", `spot: "capacity-optimized-prioritized"`, ""},
		{"spot-never", `spot: "never"`, ""},
		{"spot-unknown", `spot: "sometimes"`, "runners.test.spot"},
		{"spot-int", "spot: 1", "runners.test.spot"},
		{"spot-list", `spot: ["pco"]`, "runners.test.spot"},

		{"ssh-true", "ssh: true", ""},
		{"ssh-false", "ssh: false", ""},
		{"ssh-string-true", `ssh: "true"`, ""},
		{"ssh-string-false", `ssh: "false"`, ""},
		{"ssh-string-yes", `ssh: "yes"`, "runners.test.ssh"},
		{"ssh-int", "ssh: 1", "runners.test.ssh"},

		{"private-true", "private: true", ""},
		{"private-false", "private: false", ""},
		{"private-string-true", `private: "true"`, ""},
		{"private-string-false", `private: "false"`, ""},
		{"private-string-no", `private: "no"`, "runners.test.private"},
		{"private-int", "private: 0", "runners.test.private"},

		{"extras-single", `extras: "s3-cache"`, ""},
		{"extras-array", `extras: ["s3-cache", "ecr-cache"]`, ""},
		{"extras-plus-separated", `extras: "s3-cache+ecr-cache+efs+tmpfs"`, ""},
		{"extras-int", "extras: 1", "runners.test.extras"},
		{"extras-int-list", "extras: [1]", "runners.test.extras"},

		{"debug-true", "debug: true", ""},
		{"debug-false", "debug: false", ""},
		{"debug-string-true", `debug: "true"`, ""},
		{"debug-string-false", `debug: "false"`, ""},
		{"debug-string-on", `debug: "on"`, "runners.test.debug"},
		{"debug-int", "debug: 1", "runners.test.debug"},

		{"preinstall", "preinstall: |\n  apt-get update\n  apt-get install -y docker", ""},
		{"preinstall-list", "preinstall: [apt-get update]", "runners.test.preinstall"},

		{"prerun", "prerun: |\n  echo prepare-runner\n  systemctl restart docker", ""},
		{"prerun-list", "prerun: [echo prepare-runner]", "runners.test.prerun"},

		{"tags", `tags: ["Team:DevOps", "Environment:Production"]`, ""},
		{"tags-single", `tags: ["Team:DevOps"]`, ""},
		{"tags-map", "tags: {Team: DevOps}", "runners.test.tags"},
		{"tags-int-list", "tags: [1]", "runners.test.tags"},

		{"id", "id: custom-runner-id", ""},
		{"id-int", "id: 123", "runners.test.id"},

		{"unknown-field", "bogus: value", "runners.test.bogus"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, withFields(runnerYAML, tc.fields), tc.wantErr)
		})
	}
}

func TestValidateReader_ImageFieldsIndividually(t *testing.T) {
	testCases := []struct {
		name    string
		fields  string
		wantErr string // substring of an expected error; empty means valid
	}{
		{"preinstall", "preinstall: |\n  apt-get update\n  apt-get install -y docker", ""},
		{"prerun", "prerun: |\n  echo prepare-boot\n  systemctl restart docker", ""},
		// Resolved image metadata must not be set by users.
		{"main_disk_size", "main_disk_size: 120", "images.test.main_disk_size"},
		{"root_device_name", "root_device_name: /dev/sda1", "images.test.root_device_name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidation(t, withFields(imageYAML, tc.fields), tc.wantErr)
		})
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

// assertValidation validates yamlContent and expects no error diagnostics when
// wantErr is empty, or at least one error containing wantErr otherwise.
func assertValidation(t *testing.T, yamlContent, wantErr string) {
	t.Helper()
	diags, err := validate.ValidateReader(context.Background(), strings.NewReader(yamlContent), "test.yml")
	if err != nil {
		t.Fatalf("ValidateReader failed: %v", err)
	}
	errs := filterErrors(diags)
	if wantErr == "" {
		for _, diag := range errs {
			t.Errorf("unexpected error: %s", diag.Message)
		}
		return
	}
	for _, diag := range errs {
		if strings.Contains(diag.Message, wantErr) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got %d errors: %v", wantErr, len(errs), errs)
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}
