# Developer Guide

## Setup

### Initial Setup

```bash
# Install dependencies (Go, CUE CLI, golangci-lint)
make setup

# Or manually:
mise install
```

**Note**: Make sure mise is activated in your shell. Add this to your shell config:
```bash
eval "$(mise activate zsh)"  # for zsh
eval "$(mise activate bash)"  # for bash
```

## Project Structure

```
.
├── schema/              # CUE schema definitions
│   ├── runs_on.cue     # Main schema file
│   ├── schema.json      # Generated JSON schema
│   └── testdata/        # Test fixtures
├── pkg/
│   ├── validate/        # Go validation package
│   └── schemajson/      # JSON schema access
├── cmd/
│   └── lint/  # CLI linter binary
└── .github/
    └── workflows/      # CI/CD workflows
```

## Workflow

### Making Schema Changes

1. **Edit the CUE schema**: Modify `schema/runs_on.cue`
2. **Test locally**: Run `make test` to ensure tests pass
3. **Generate JSON schema**: Run `make gen` to update `schema/schema.json`
4. **Commit both files**: Always commit both `.cue` and `.json` files together

### Adding New Fields

When adding new fields to the schema:

1. Update `schema/runs_on.cue` with the new field definition
2. Add test cases in `schema/testdata/valid/` and `schema/testdata/invalid/`
3. Update the Go types if needed (in the main runs-on repo)
4. Regenerate JSON schema: `make gen`
5. Update documentation in `README.md`

### Versioning

- Follow semantic versioning (v0.1.0, v0.2.0, etc.)
- Tag releases in git
- Update version strings in:
  - `cmd/lint/main.go`
  - `cli/internal/cli/lint.go` (SARIF output)

## Testing

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./pkg/validate/...

# With verbose output
go test -v ./pkg/validate/...
```

### Adding Test Cases

1. **Valid configs**: Add to `schema/testdata/valid/`
2. **Invalid configs**: Add to `schema/testdata/invalid/`
3. **Update test file**: Add test cases in `pkg/validate/validator_test.go`

## Dependencies

### Adding Dependencies

```bash
go get <package>
go mod tidy
```

### Updating Dependencies

```bash
go get -u <package>
go mod tidy
```

## CI/CD

The GitHub Actions workflow (`.github/workflows/ci.yml`) runs:

1. Go tests
2. Schema generation check (ensures `schema.json` is up to date)
3. Linting with golangci-lint

## Integration with RunsOn CLI

The CLI integration uses a `replace` directive during development:

```go
//go:build dev
// +build dev

replace github.com/runs-on/config => ../config
```

For production, remove the replace directive and use the published module.

## Troubleshooting

### Schema Not Found

If you get "schema file not found" errors:

1. Ensure `schema/runs_on.cue` exists
2. Check that the embed path is correct: `//go:embed ../../schema/runs_on.cue`
3. Verify you're running from the correct directory

### CUE Compilation Errors

If CUE schema fails to compile:

1. Check CUE syntax: `cue vet schema/runs_on.cue`
2. Verify all definitions are properly closed
3. Check for circular references

### YAML Anchor Issues

YAML anchors are handled automatically by the YAML parser. If you see issues:

1. Verify anchor syntax is correct
2. Check that aliases reference existing anchors
3. Ensure anchors are defined before use


