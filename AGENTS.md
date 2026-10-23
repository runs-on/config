# Config guide

Rules for the `runs-on.yml` CUE schema and validator module, published from
`config/` in `runs-on/monorepo` to `runs-on/config`. Edit it and open pull
requests in the monorepo, never in the mirror; `../` paths and root `make`
targets exist only there.

## Rules

- Edit only `schema/runs_on.cue`, then run `make gen`. It overwrites
  `schema/schema.json`, `pkg/schemajson/schema.json`, and
  `pkg/validate/schema.cue`, and CI fails when they are stale.
- Pair each schema change with `schema/testdata/valid/` or `invalid/` files.
  List each new file in `TestValidateFile_Valid` or `TestValidateFile_Invalid`
  (`pkg/validate/validator_test.go`); nothing discovers them on its own.
- Keep each runner, image, pool, schedule, and match field in sync with its Go
  type (`../pkg/specs`, `../pkg/config`, `../pkg/pool`, `../pkg/fleet/config`)
  and with `variables.tf` and `README.md` in `../terraform/modules/fleet/`: run
  `make check-fleet-catalog-contract` at the repo root (`make test` here skips
  it).
- `pkg/validate` is a public API that also gates `roc lint`, Flex repo configs
  (logged only), and the Fleet Terraform catalog, which fleetd refuses to load
  on any diagnostic, warnings included (`validateWithSharedSchema` in
  `../pkg/fleet/config/config.go`). Treat a tighter constraint or a new warning
  as a breaking change.
- Pool names are the `pools` map keys; `#PoolSpec` has no `name` field. The key
  pattern is repeated in `../pkg/pool/standby/pool.go`.
- Document field changes in `README.md` and in `runs-on/runs-on.com`:
  `src/content/docs/runners/labels.mdx` (Repository configuration) and, for
  pools, `src/content/docs/performance/warm-pools.mdx`.
- Never hand-edit the `rev:` pins in `README.md` or the Versioning section of
  `DEVELOPMENT.md`; `make sync-metadata` at the repo root rewrites them.

## Commands

- `make setup` once: it installs the `cue` CLI that `make gen` runs.
