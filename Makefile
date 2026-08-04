.PHONY: help test test-coverage lint fmt vet examples clean

# Default target
help:
	@echo "Available targets:"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make examples       - Run all examples"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install-deps   - Install development dependencies"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run all examples
examples:
	@echo "Running basic example..."
	cd examples/basic && go run main.go
	@echo ""
	@echo "Running custom example..."
	cd examples/custom && go run main.go
	@echo ""
	@echo "Running advanced example..."
	cd examples/advanced && go run main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f coverage.out coverage.html
	go clean -cache -testcache

# Install development dependencies
install-deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Build examples
build-examples:
	@echo "Building examples..."
	cd examples/basic && go build -o basic main.go
	cd examples/otlp-http && go build -o otlp-http main.go
	cd examples/otlp-grpc && go build -o otlp-grpc main.go
	cd examples/prometheus && go build -o prometheus main.go
	cd examples/custom && go build -o custom main.go
	cd examples/advanced && go build -o advanced main.go
	@echo "Examples built successfully"
