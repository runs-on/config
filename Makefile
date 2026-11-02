MONOREPO_ROOT := ..
VERSION ?= $(shell if [ -f ../VERSION ]; then tr -d '\n' < ../VERSION; elif [ -f VERSION ]; then tr -d '\n' < VERSION; elif git describe --tags --exact-match >/dev/null 2>&1; then git describe --tags --exact-match; else echo dev; fi)
LDFLAGS = -X github.com/runs-on/config/internal/version.Version=$(VERSION)

.PHONY: gen lint test install clean sync-schema setup sync-metadata version

setup:
	@echo "Installing dependencies with mise..."
	mise install
	@echo "Installing CUE CLI..."
	GOSUMDB=sum.golang.org mise exec -- go install cuelang.org/go/cmd/cue@latest
	@echo "Setup complete! Run 'make gen' to generate schema files."
	@echo "Note: Make sure mise is activated in your shell (run 'mise activate' or add to your shell config)"

gen:
	@echo "Generating JSON schema from CUE..."
	rm -f schema/schema.json pkg/schemajson/schema.json
	cd schema && mise exec -- go generate
	@echo "Copying schema.json to pkg/schemajson..."
	cp schema/schema.json pkg/schemajson/schema.json
	@echo "Syncing schema.cue to pkg/validate..."
	cp schema/runs_on.cue pkg/validate/schema.cue

sync-schema:
	@echo "Syncing schema.cue to pkg/validate..."
	cp schema/runs_on.cue pkg/validate/schema.cue

lint:
	@$(MAKE) -C $(MONOREPO_ROOT) lint-config-module

test:
	@echo "Running tests..."
	mise exec -- go test ./...

install:
	@echo "Installing lint..."
	mise exec -- go install -ldflags "$(LDFLAGS)" ./cmd/lint

clean:
	@echo "Cleaning generated files..."
	rm -f schema/schema.json pkg/schemajson/schema.json

sync-metadata:
	@VERSION="$(VERSION)" perl -0pi -e 's{rev: v[0-9A-Za-z.\\-]+}{rev: $$ENV{VERSION}}g' README.md

version:
	@echo $(VERSION)
