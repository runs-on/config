.PHONY: gen lint test install clean sync-schema

gen:
	@echo "Generating JSON schema from CUE..."
	@cd schema && mise exec -- go generate
	@echo "Syncing schema.cue to pkg/validate..."
	@cp schema/runs_on.cue pkg/validate/schema.cue

sync-schema:
	@echo "Syncing schema.cue to pkg/validate..."
	@cp schema/runs_on.cue pkg/validate/schema.cue

lint:
	@echo "Running golangci-lint..."
	@mise exec -- golangci-lint run

test:
	@echo "Running tests..."
	@mise exec -- go test ./...

install:
	@echo "Installing runs-on-config-lint..."
	@mise exec -- go install ./cmd/runs-on-config-lint

clean:
	@echo "Cleaning generated files..."
	@rm -f schema/schema.json

