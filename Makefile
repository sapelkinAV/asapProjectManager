.PHONY: all build test lint format clean deps install

# Default target
all: build

# Build the binary
build:
	go build -o asapm .

# Run tests
test:
	go test ./...

# Lint: check formatting and run static analysis
lint:
	gofmt -d .
	go vet ./...

# Format code
format:
	gofmt -w .

# Clean build artifacts
clean:
	rm -f asapm

# Tidy dependencies
deps:
	go mod tidy

# Install to ./local/bin
install: build
	mkdir -p ~/.local/bin
	cp asapm ~/.local/bin/