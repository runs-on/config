# AI Agent Guide

This guide helps AI agents understand how to work with this codebase. For detailed developer information, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## Quick Reference

### Key Files
- **Schema**: `schema/runs_on.cue` (main) and `pkg/validate/schema.cue` (copy for validation)
- **Tests**: `pkg/validate/validator_test.go`
- **Test Data**: `schema/testdata/valid/` and `schema/testdata/invalid/`
- **JSON Schema**: `schema/schema.json` (generated, don't edit manually)

### Critical Rules

1. **Always update both schema files**: When modifying the schema, update BOTH:
   - `schema/runs_on.cue` (source of truth)
   - `pkg/validate/schema.cue` (used by validator)

2. **After schema changes**: Run `make gen` to regenerate `schema/schema.json`

3. **Test files**: When adding/modifying schema, update corresponding test files in `schema/testdata/`

## Common Tasks

### Adding a New Field to PoolSpec/RunnerSpec/ImageSpec

1. Edit `schema/runs_on.cue` - add field definition
2. Edit `pkg/validate/schema.cue` - add same field definition
3. Add test cases:
   - Valid case: `schema/testdata/valid/`
   - Invalid case: `schema/testdata/invalid/`
4. Update tests in `pkg/validate/validator_test.go` if needed
5. Run `make gen` to regenerate JSON schema
6. Run `make test` to verify

### Removing a Field

1. Remove from both `schema/runs_on.cue` and `pkg/validate/schema.cue`
2. Remove field from all test files in `schema/testdata/`
3. Remove/update related tests in `pkg/validate/validator_test.go`
4. Run `make gen` and `make test`

### Modifying Validation Rules

1. Update constraints in both CUE schema files
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

- **Pool names**: The `name` field was removed from `#PoolSpec`. Pool names are derived from the pool key in the YAML.
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

## When Making Changes

1. **Read the existing code** - understand patterns before modifying
2. **Update both schemas** - `schema/runs_on.cue` AND `pkg/validate/schema.cue`
3. **Add tests** - both valid and invalid cases
4. **Regenerate JSON** - run `make gen`
5. **Verify** - run `make test`
6. **Check lints** - run `make lint`

## Common Pitfalls

- ❌ Editing only one schema file (must edit both)
- ❌ Forgetting to run `make gen` after schema changes
- ❌ Not updating test files when removing fields
- ❌ Adding fields without tests
- ❌ Manually editing `schema.json` (it's generated)

## Reference

- [DEVELOPMENT.md](./DEVELOPMENT.md) - Full developer guide
- [README.md](./README.md) - User-facing documentation

