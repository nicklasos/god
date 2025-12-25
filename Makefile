.PHONY: help build install test test-verbose coverage clean run fmt lint vet release snapshot

# Binary name
BINARY_NAME=god

# Default target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make install       - Install the binary to GOPATH/bin"
	@echo "  make test          - Run tests"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make run           - Build and run the application"
	@echo "  make fmt           - Format code with gofmt"
	@echo "  make lint          - Run golangci-lint (if installed)"
	@echo "  make vet           - Run go vet"
	@echo "  make check         - Run fmt, vet, and test"
	@echo "  make release       - Create a release with GoReleaser"
	@echo "  make snapshot      - Create a snapshot build with GoReleaser"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install

# Run tests
test:
	@echo "Running tests..."
	@go test ./internal/...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./internal/...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./internal/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf bin/ dist/

# Build and run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run golangci-lint (if installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: brew install golangci-lint"; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run all checks
check: fmt vet test
	@echo "All checks passed!"

# Create a release with GoReleaser (requires git tag)
release:
	@echo "Creating release with GoReleaser..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --clean; \
	else \
		echo "GoReleaser not installed. Install with: brew install goreleaser"; \
	fi

# Create a snapshot build (no git tag required)
snapshot:
	@echo "Creating snapshot build..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser build --snapshot --clean; \
	else \
		echo "GoReleaser not installed. Install with: brew install goreleaser"; \
	fi

