# Developer Guide

## Setup

### Initial Setup

```bash
# Install dependencies
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
└── cmd/
    └── lint/            # CLI linter binary
```

## Workflow

### Making Schema Changes

1. **Edit the CUE schema**: Modify `schema/runs_on.cue`
2. **Regenerate derived files**: Run `make gen`. It writes `schema/schema.json` and `pkg/schemajson/schema.json`, and copies the schema to `pkg/validate/schema.cue`, which the validator embeds
3. **Test locally**: Run `make test` to ensure tests pass
4. **Commit the generated files**: Commit them together with `schema/runs_on.cue`

### Adding New Fields

When adding new fields to the schema:

1. Update `schema/runs_on.cue` with the new field definition
2. Add test cases in `schema/testdata/valid/` and `schema/testdata/invalid/`
3. Update the Go types if needed. In the monorepo they live in `../pkg/specs`,
   `../pkg/config`, `../pkg/pool` and `../pkg/fleet/config`. Fleet fields are also
   declared in `../terraform/modules/fleet/variables.tf` and documented in
   `../terraform/modules/fleet/README.md`; run `make check-fleet-catalog-contract`
   at the monorepo root to check them against `schema/runs_on.cue`
4. Regenerate JSON schema: `make gen`
5. Update documentation in `README.md`

### Versioning

- Release version comes from the repository `VERSION` file in the monorepo, or the mirrored repo `VERSION` file after export.
- Release tags must match `VERSION`.
- Do not hand-edit version strings in code; sync metadata from `VERSION` before tagging.

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

The monorepo workflow `.github/workflows/config-test.yml` runs:

1. `make gen`, then fails if any of `schema/schema.json`,
   `pkg/schemajson/schema.json`, `pkg/validate/schema.cue` or
   `cue.mod/module.cue` differs from the committed copy
2. Go tests via `make test`
3. Linting with the monorepo root config via `make lint`

## Integration with RunsOn CLI

In the monorepo, the root module resolves this directory with
`replace github.com/runs-on/config => ./config` in `../go.mod`, so root checks
that run with `GOWORK=off` still build against local changes. The CLI in
`../cli` picks up `./config` through the root `go.work` instead. `cli/go.mod`
must not add a `replace`, because it is mirrored to `runs-on/cli` as a
standalone module; `scripts/check-go-workspace-scope.sh` at the monorepo root
enforces this.

## Troubleshooting

### Stale Embedded Schema

The validator embeds `pkg/validate/schema.cue`, not `schema/runs_on.cue`. If
validation ignores a schema change, run `make gen` to copy the schema there.

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
