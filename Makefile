# Basic project information
BINARY_NAME := aws-bia
VERSION := 0.1.0
BUILD_DIR := build

# Go parameters
GO := go
GOFMT := gofmt
GOTEST := go test
GOVET := go vet
GOLINT := golangci-lint
GOBUILD := $(GO) build
GOINSTALL := $(GO) install
GOCLEAN := $(GO) clean
GOGET := $(GO) get

# Build flags
LDFLAGS := -ldflags "-X github.com/your-username/aws-bia/cmd.Version=$(VERSION)"

# Source files
SRC_FILES := $(shell find . -type f -name "*.go" -not -path "./vendor/*")
TEST_FILES := $(shell find . -type f -name "*_test.go" -not -path "./vendor/*")

# Default target
.PHONY: all
all: lint test build

# Build the application
.PHONY: build
build:
    @echo "Building $(BINARY_NAME)..."
    @mkdir -p $(BUILD_DIR)
    $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install the application
.PHONY: install
install:
    @echo "Installing $(BINARY_NAME)..."
    $(GOINSTALL) $(LDFLAGS) .

# Clean build artifacts
.PHONY: clean
clean:
    @echo "Cleaning..."
    @rm -rf $(BUILD_DIR)
    $(GOCLEAN)

# Run tests
.PHONY: test
test:
    @echo "Running tests..."
    $(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
    @echo "Running tests with coverage..."
    $(GOTEST) -v -coverprofile=coverage.out ./...
    $(GO) tool cover -html=coverage.out -o coverage.html

# Format code
.PHONY: fmt
fmt:
    @echo "Formatting code..."
    $(GOFMT) -s -w $(SRC_FILES)

# Lint code
.PHONY: lint
lint:
    @echo "Linting code..."
    $(GOVET) ./...
    @if command -v $(GOLINT) > /dev/null; then \
        $(GOLINT) run ./...; \
    else \
        echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
    fi

# Check dependencies and update
.PHONY: deps
deps:
    @echo "Checking dependencies..."
    $(GO) mod tidy
    $(GO) mod verify

# Create a new release
.PHONY: release
release: clean test lint build
    @echo "Creating release v$(VERSION)..."
    @mkdir -p releases
    @cp $(BUILD_DIR)/$(BINARY_NAME) releases/$(BINARY_NAME)-$(VERSION)
    @echo "Release created at releases/$(BINARY_NAME)-$(VERSION)"

# Cross-compile for multiple platforms
.PHONY: cross-compile
cross-compile: clean
    @echo "Cross-compiling for multiple platforms..."
    @mkdir -p $(BUILD_DIR)
    
    # Linux (amd64)
    GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
    
    # macOS (amd64)
    GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
    
    # macOS (arm64) for M1/M2 Macs
    GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
    
    # Windows (amd64)
    GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Run the application (for development)
.PHONY: run
run:
    @echo "Running $(BINARY_NAME)..."
    $(GO) run main.go $(ARGS)

# Example help command
.PHONY: help
help:
    @echo "Available targets:"
    @echo "  all            - Run lint, test, and build"
    @echo "  build          - Build the application"
    @echo "  clean          - Clean build artifacts"
    @echo "  test           - Run tests"
    @echo "  test-coverage  - Run tests with coverage report"
    @echo "  fmt            - Format code"
    @echo "  lint           - Run linters"
    @echo "  deps           - Check and update dependencies"
    @echo "  install        - Install the application"
    @echo "  release        - Create a new release"
    @echo "  cross-compile  - Build for multiple platforms"
    @echo "  run            - Run the application (for development)"
    @echo "  help           - Show this help message"
    @echo ""
    @echo "Example usage:"
    @echo "  make build"
    @echo "  make run ARGS=\"invoke --help\""