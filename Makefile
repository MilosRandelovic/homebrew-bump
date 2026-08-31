MCP_PACKAGE=github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp
MCP_VERSION=v2.2.0

.PHONY: build clean smoke install help

# Default target
all: build

# Build the CLI and MCP server
build:
	go build -o bump
	GOBIN="$(CURDIR)" go install $(MCP_PACKAGE)@$(MCP_VERSION)

# Clean build artifacts
clean:
	rm -f bump bump-mcp

# Run CLI smoke tests
smoke: build
	./scripts/smoke-test.sh ./bump ./bump-mcp

# Install dependencies
deps:
	go mod tidy

# Install both binaries to GOPATH/bin
install: build
	go install
	go install $(MCP_PACKAGE)@$(MCP_VERSION)

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  clean    - Clean build artifacts"
	@echo "  smoke    - Build and run CLI smoke tests"
	@echo "  deps     - Install dependencies"
	@echo "  install  - Install binary to GOPATH/bin"
	@echo "  help     - Show this help"
