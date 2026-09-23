MCP_PACKAGE := github.com/MilosRandelovic/bump-core/v2/cmd/bump-mcp
ACTIONLINT_VERSION := v1.7.12

.PHONY: all build clean smoke deps install help workflow-lint

all: build

build:
	go build -o bump
	go build -o bump-mcp $(MCP_PACKAGE)

clean:
	rm -f bump bump-mcp

smoke: build
	./scripts/smoke-test.sh ./bump ./bump-mcp

deps:
	go mod tidy

install: build
	go install
	go install $(MCP_PACKAGE)

workflow-lint:
	go run github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION) .github/workflows/*.yml

help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  smoke         - Build and run CLI smoke tests"
	@echo "  deps          - Install dependencies"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  workflow-lint - Validate GitHub Actions workflows"
	@echo "  help          - Show this help"
