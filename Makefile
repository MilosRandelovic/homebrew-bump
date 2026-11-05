MCP_PACKAGE := github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp
ACTIONLINT_VERSION := v1.7.12

.PHONY: all build clean smoke test-smoke deps install help workflow-lint

# Default target
all: build

# Build the CLI and MCP server
build:
	go build -o bump
	go build -mod=mod -o bump-mcp $(MCP_PACKAGE)

# Clean build artifacts
clean:
	rm -f bump bump-mcp

# Run CLI smoke tests
smoke: build
	./scripts/smoke-test.sh ./bump ./bump-mcp

test-smoke:
	./scripts/smoke-test-test.sh

# Install dependencies
deps:
	go mod tidy

# Install both binaries to GOPATH/bin
install: build
	go install
	go install $(MCP_PACKAGE)

workflow-lint:
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION) .github/workflows/*.yml

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  smoke         - Build and run CLI smoke tests"
	@echo "  test-smoke    - Test smoke-check guards with fixtures"
	@echo "  deps          - Install dependencies"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  workflow-lint - Validate GitHub Actions workflows"
	@echo "  help          - Show this help"
