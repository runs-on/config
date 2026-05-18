package repo

import (
	"context"
	"reflect"
	"testing"
)

func TestParseInitializesEmptyMaps(t *testing.T) {
	repoConfig, err := Parse("admins:\n  - alice\n")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if repoConfig.Runners == nil || repoConfig.Images == nil || repoConfig.Pools == nil {
		t.Fatalf("expected maps to be initialized, got %+v", repoConfig)
	}
	if !reflect.DeepEqual(repoConfig.Admins, []string{"alice"}) {
		t.Fatalf("Admins = %v, want [alice]", repoConfig.Admins)
	}
}

func TestRepoConfigYAMLRoundTrip(t *testing.T) {
	original := RepoConfig{
		Extends: "shared-repo",
		Runners: map[string]RunnerSpec{
			"test-runner": {
				Image:  "ubuntu22-full-x64",
				Cpu:    []int{2},
				Ram:    []int{8},
				Family: []string{"c7a"},
			},
		},
		Images: map[string]ImageSpec{
			"test-image": {
				Ami:      "ami-1234567890abcdef0",
				Platform: "linux",
				Arch:     "x64",
			},
		},
		Pools: map[string]PoolSpec{
			"default": {
				Schedule: []PoolSchedule{},
				Runner:   "test-runner",
			},
		},
		Admins: []string{"alice"},
	}

	dump, err := original.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error = %v", err)
	}

	roundTripped, err := Parse(dump)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !reflect.DeepEqual(original, roundTripped) {
		t.Fatalf("round trip mismatch:\n got: %+v\nwant: %+v", roundTripped, original)
	}
}

func TestValidateReportsDiagnostics(t *testing.T) {
	diags, err := Validate(context.Background(), "runners: [\n", "runs-on.yml")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	if diags[0].Path != "runs-on.yml" {
		t.Fatalf("diagnostic path = %q, want runs-on.yml", diags[0].Path)
	}
}

func TestDiagnosticsErrorFormatsLocations(t *testing.T) {
	err := DiagnosticsError("bad config", []Diagnostic{
		{Path: "runs-on.yml", Line: 3, Column: 7, Message: "missing field"},
		{Path: "runs-on.yml", Message: "top-level problem"},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	want := "bad config: runs-on.yml:3:7: missing field; runs-on.yml: top-level problem"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
