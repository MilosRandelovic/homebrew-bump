# Bump Makefile

.PHONY: build clean test install help

# Default target
all: build

# Build the application
build:
	go build -o bump

# Clean build artifacts
clean:
	rm -f bump

# Run tests
test:
	go test ./...

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
	@echo "  test     - Run tests"
	@echo "  deps     - Install dependencies"
	@echo "  install  - Install binary to GOPATH/bin"
	@echo "  help     - Show this help"
