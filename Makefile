.PHONY: gen lint test install clean sync-schema setup update-dependents

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
	@echo "Running golangci-lint..."
	mise exec -- golangci-lint run

test:
	@echo "Running tests..."
	mise exec -- go test ./...

install:
	@echo "Installing lint..."
	mise exec -- go install ./cmd/lint

clean:
	@echo "Cleaning generated files..."
	rm -f schema/schema.json pkg/schemajson/schema.json

update-dependents:
	@echo "Updating dependents to latest config module..."
	@SHA=$$(git rev-parse HEAD); \
	echo "Using commit SHA: $$SHA"; \
	if [ -d "../cli" ]; then \
		echo "Updating ../config..."; \
		cd ../cli && mise exec -- go get -u github.com/runs-on/config@$$SHA && mise exec -- go mod tidy; \
	else \
		echo "Warning: ../cli not found, skipping"; \
	fi; \
	if [ -d "../runs-on/server" ]; then \
		echo "Updating ../config/server..."; \
		cd ../runs-on/server && mise exec -- go get -u github.com/runs-on/config@$$SHA && mise exec -- go mod tidy; \
	else \
		echo "Warning: ../runs-on/server not found, skipping"; \
	fi
	@echo "Done updating dependents"

