.PHONY: build install clean test run deps

BINARY_NAME=safeshell
VERSION=0.1.0
BUILD_DIR=./build
INSTALL_DIR=$(HOME)/go/bin

# Build flags
LDFLAGS=-ldflags "-X github.com/safeshell/safeshell/internal/cli.version=$(VERSION)"

# Default target
all: build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build the binary
build: deps
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/safeshell

# Install to GOPATH/bin
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Run 'safeshell init' to set up shell aliases"

# Install locally to /usr/local/bin (requires sudo)
install-local: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

# Run without installing
run: build
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests (short output)
test-short:
	go test ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(HOME)/.safeshell/checkpoints/*

# Development: build and run
dev: build
	$(BUILD_DIR)/$(BINARY_NAME) status

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "SafeShell Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build      - Build the binary"
	@echo "  make install    - Install to ~/go/bin"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make deps       - Download dependencies"
	@echo "  make fmt        - Format code"
	@echo "  make run ARGS=  - Run with arguments"
	@echo ""
	@echo "Examples:"
	@echo "  make run ARGS='status'"
	@echo "  make run ARGS='list'"
