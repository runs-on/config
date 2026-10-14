# AI Agent Guide

This guide helps AI agents understand how to work with this codebase. For detailed developer information, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## Project Context

- Read `../docs/PROJECT_CONTEXT.md` first for product/business context and customer-facing invariants.
- Schema and validator changes affect the documented `.github/runs-on.yml` contract. Before changing field names or semantics, compare against `../../marketing/src/content/docs/configuration/repo-config.mdx` and the related public docs linked from `../docs/PROJECT_CONTEXT.md`.

## Quick Reference

### Key Files
- **Schema**: `schema/runs_on.cue` (source of truth); `make gen` copies it to `pkg/validate/schema.cue` (embedded by the validator)
- **Tests**: `pkg/validate/validator_test.go`
- **Test Data**: `schema/testdata/valid/` and `schema/testdata/invalid/`
- **JSON Schema**: `schema/schema.json` (generated, don't edit manually)

### Rules

- Edit only `schema/runs_on.cue`, then run `make gen`: it regenerates `schema/schema.json` and `pkg/schemajson/schema.json` and overwrites `pkg/validate/schema.cue`, so hand edits to any of those are lost.
- Schema changes come with matching cases in `schema/testdata/valid/` and `schema/testdata/invalid/`.

## Common Tasks

### Adding a New Field to PoolSpec/RunnerSpec/ImageSpec

1. Edit `schema/runs_on.cue` - add field definition
2. Add test cases:
   - Valid case: `schema/testdata/valid/`
   - Invalid case: `schema/testdata/invalid/`
3. Update tests in `pkg/validate/validator_test.go` if needed
4. Run `make gen` to regenerate the derived schema files
5. Run `make test` to verify

### Removing a Field

1. Remove from `schema/runs_on.cue`
2. Remove field from all test files in `schema/testdata/`
3. Remove/update related tests in `pkg/validate/validator_test.go`
4. Run `make gen` and `make test`

### Modifying Validation Rules

1. Update constraints in `schema/runs_on.cue` and run `make gen`
2. Add/update test cases to verify the new rules
3. Update tests in `validator_test.go`
4. Run `make test` to ensure existing tests still pass

## Schema Structure

The schema defines:
- `#RepoConfig`: Top-level config structure
- `#RunnerSpec`: Runner configuration
- `#ImageSpec`: Image configuration
- `#PoolSpec`: Pool configuration (name comes from pool key, not a field)
- `#PoolSchedule`: Schedule entries within pools

## Testing Patterns

### Valid Config Test
```go
func TestValidateFile_NewFeature(t *testing.T) {
    testFile := "../../schema/testdata/valid/new-feature.yml"
    diags, err := validate.ValidateFile(context.Background(), testFile)
    // ... verify no errors
}
```

### Invalid Config Test
```go
func TestValidateFile_InvalidFeature(t *testing.T) {
    testFile := "../../schema/testdata/invalid/invalid-feature.yml"
    diags, err := validate.ValidateFile(context.Background(), testFile)
    // ... verify errors are present
}
```

## Important Notes

- **Pool names**: Pool names come from the pool key in the YAML; `#PoolSpec` has no `name` field.
- **Optional fields**: Use `field?: type` syntax
- **Required fields**: Use `field: type` syntax (no `?`)
- **Constraints**: Add with `&` operator, e.g., `name?: string & != "" & =~"^[a-z0-9_-]+$"`
- **Custom fields**: Top-level custom fields are allowed (prefixed with `x-` recommended)

## Commands

```bash
make test      # Run all tests
make gen       # Regenerate schema.json
make lint      # Run linter
make setup     # Install dependencies
```

## Reference

- [DEVELOPMENT.md](./DEVELOPMENT.md) - Full developer guide
- [README.md](./README.md) - User-facing documentation
