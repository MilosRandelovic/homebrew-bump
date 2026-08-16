# Bump Makefile

.PHONY: build clean smoke install help

# Default target
all: build

# Build the application
build:
	go build -o bump

# Clean build artifacts
clean:
	rm -f bump

# Run CLI smoke tests
smoke: build
	./scripts/smoke-test.sh ./bump

# Install dependencies
deps:
	go mod tidy

# Install the binary to GOPATH/bin
install: build
	go install

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  clean    - Clean build artifacts"
	@echo "  smoke    - Build and run CLI smoke tests"
	@echo "  deps     - Install dependencies"
	@echo "  install  - Install binary to GOPATH/bin"
	@echo "  help     - Show this help"
