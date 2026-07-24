package schema

// The -e '#RepoConfig' selector is required: without it cue exports the
// package's (empty) concrete value and the schema degenerates to a bare
// open object with no properties.
//go:generate cue export runs_on.cue --out=jsonschema -e #RepoConfig --outfile=schema.json
