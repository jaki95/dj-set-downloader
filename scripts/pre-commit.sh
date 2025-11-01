#!/bin/bash

if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

echo "Running go fmt..."
go fmt ./...
if [ $? -ne 0 ]; then
    echo "Error: go fmt failed"
    exit 1
fi

echo "Running go vet..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "Error: go vet failed"
    exit 1
fi

echo "Running golangci-lint..."
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
if [ $? -ne 0 ]; then
    echo "Error: golangci-lint found issues"
    exit 1
fi

echo "Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "Error: tests failed"
    exit 1
fi

echo "All pre-commit checks passed!"
exit 0 