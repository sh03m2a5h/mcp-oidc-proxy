.PHONY: all build test clean lint run docker help

# Variables
BINARY_NAME := mcp-oidc-proxy
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version.Version=$(VERSION) \
	-X github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version.BuildDate=$(BUILD_DATE)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet
GOFMT := gofmt

# Directories
CMD_DIR := ./cmd/mcp-oidc-proxy
BIN_DIR := bin
COVERAGE_DIR := coverage

## help: Show this help message
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Run tests and build the binary
all: test build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOCMD) tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## lint: Run linters
lint:
	@echo "Running linters..."
	$(GOVET) ./...
	@if [ -x "$$(command -v golangci-lint)" ]; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, installing v1.55.2..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2; \
		$$(go env GOPATH)/bin/golangci-lint run; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w -s .

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR) coverage.out

## run: Run the application
run: build
	./$(BIN_DIR)/$(BINARY_NAME)

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -f deployments/docker/Dockerfile .

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## build-all: Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	# Darwin ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Multi-platform build complete"

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"