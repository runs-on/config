# RunsOn Config Schema

This repository contains the schema definition and validation tools for `runs-on.yml` configuration files used by RunsOn.

## Overview

The RunsOn config schema defines the structure and validation rules for repository configuration files. This module provides:

- **CUE Schema**: Authoritative schema definition in CUE format (`schema/runs_on.cue`)
- **JSON Schema**: Generated JSON schema for tooling integration (`schema/schema.json`)
- **Go Validation Library**: Go package for validating config files (`pkg/validate`)
- **CLI Linter**: Standalone binary for linting config files (`cmd/runs-on-config-lint`)

## Installation

### Go Library

```bash
go get github.com/runs-on/config
```

### CLI Linter

```bash
go install github.com/runs-on/config/cmd/runs-on-config-lint@latest
```

## Usage

### Go Library

```go
import (
    "context"
    "github.com/runs-on/config/pkg/validate"
)

diagnostics, err := validate.ValidateFile(ctx, "path/to/runs-on.yml")
if err != nil {
    // handle error
}

for _, diag := range diagnostics {
    fmt.Printf("%s:%d:%d: %s\n", diag.Path, diag.Line, diag.Column, diag.Message)
}
```

### CLI Linter

```bash
# Validate a file
runs-on-config-lint path/to/runs-on.yml

# Read from stdin
cat runs-on.yml | runs-on-config-lint --stdin

# JSON output
runs-on-config-lint --format json path/to/runs-on.yml

# SARIF output (for GitHub Actions)
runs-on-config-lint --format sarif path/to/runs-on.yml
```

### RunsOn CLI Integration

The `roc` CLI includes a `lint` command:

```bash
roc lint path/to/runs-on.yml
roc lint --format json path/to/runs-on.yml
roc lint --stdin < runs-on.yml
```

## Schema Structure

The `runs-on.yml` file supports:

- `_extends`: Reference to another repository's config (string)
- `runners`: Map of runner specifications
- `images`: Map of image specifications  
- `pools`: Map of pool specifications
- `admins`: List of admin usernames (array of strings)

### Runner Specification

```yaml
runners:
  my-runner:
    cpu: [2, 4]           # CPU count(s) - int, string, or array
    ram: [16, 32]         # RAM in GB - int, string, or array
    family: [c7a, m7a]    # Instance family
    image: ubuntu22-full-x64
    spot: "pco"           # Spot configuration
    ssh: false             # SSH access (bool or string)
    private: true          # Private network (bool or string)
    volume: "80gb:gp3:125mbs:3000iops"  # Volume spec
    extras: ["s3-cache"]   # Extra features
    tags: ["Team:DevOps"]  # Tags
```

### Image Specification

```yaml
images:
  ubuntu22-custom:
    ami: ami-1234567890abcdef0
    platform: linux
    arch: x64
    name: ubuntu-22.04
    owner: 123456789012
    preinstall: |
      apt-get update
      apt-get install -y docker
```

### Pool Specification

```yaml
pools:
  dependabot:
    name: dependabot
    env: production
    timezone: UTC
    runner: small-x64
    max_surge: 5
    schedule:
      - name: default
        hot: 2
        stopped: 3
      - name: nights
        hot: 0
        stopped: 1
        match:
          day: [monday, tuesday]
          time: ["22:00", "06:00"]
```

## YAML Anchors Support

The validator fully supports YAML anchors and aliases:

```yaml
runners:
  base-runner: &base
    cpu: [2]
    ram: [16]
    family: [c7a]

  extended-runner:
    <<: *base
    cpu: [4]
```

## Development

### Updating the Schema

1. Edit `schema/runs_on.cue`
2. Run `make gen` to regenerate `schema/schema.json`
3. Run `make test` to verify changes
4. Commit both `.cue` and `.json` files

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

### Building the CLI

```bash
make install
# or
go install ./cmd/runs-on-config-lint
```

## CI Integration

### GitHub Actions

Add this to your workflow:

```yaml
- name: Validate runs-on.yml
  uses: docker://ghcr.io/runs-on/config-lint:latest
  with:
    args: .github/runs-on.yml
```

Or use the binary:

```yaml
- name: Install linter
  run: go install github.com/runs-on/config/cmd/runs-on-config-lint@latest

- name: Validate config
  run: runs-on-config-lint .github/runs-on.yml
```

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/runs-on/config
    rev: v0.1.0
    hooks:
      - id: runs-on-config-lint
        args: [--format, json]
```

## License

MIT
