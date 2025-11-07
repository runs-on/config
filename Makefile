.PHONY: gen lint test install clean

gen:
	@echo "Generating JSON schema from CUE..."
	@cd schema && mise exec -- go generate

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

