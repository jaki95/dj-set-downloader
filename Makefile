.PHONY: help build test clean swagger-docs python-client setup-dev install-deps run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development setup
setup-dev: install-deps ## Set up development environment
	@echo "Setting up development environment..."
	@echo "Development environment setup complete!"

install-deps: ## Install all required dependencies
	@echo "Installing dependencies..."
	@echo "Installing Go dependencies..."
	@go mod download
	@echo "Installing swag for API documentation..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Installing OpenAPI Generator..."
	@npm install -g @openapitools/openapi-generator-cli
	@echo "Installing Python client dependencies..."
	@if command -v uv >/dev/null 2>&1; then \
		cd clients/python && uv sync; \
	else \
		echo "uv not found, skipping Python dependencies"; \
		echo "Install uv: curl -LsSf https://astral.sh/uv/install.sh | sh"; \
	fi

# Build targets
build: ## Build the Go application
	@echo "Building Go application..."
	@go build -o bin/dj-set-downloader cmd/main.go
	@echo "Build complete: bin/dj-set-downloader"

run: build ## Build and run the application
	@echo "Starting DJ Set Downloader..."
	@./bin/dj-set-downloader

# Documentation and client generation
swagger-docs: ## Generate Swagger documentation from Go annotations
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/main.go -o docs/
	@echo "Swagger documentation generated!"
	@echo "Files updated: docs/swagger.json, docs/swagger.yaml, docs/docs.go"

python-client: swagger-docs ## Generate Python client from OpenAPI spec
	@echo "Generating Python client..."
	@openapi-generator generate \
		-i docs/swagger.json \
		-g python \
		-o ./clients/python \
		-c ./clients/openapi-generator-config.yaml
	@echo "Installing Python client dependencies..."
	@if command -v uv >/dev/null 2>&1; then \
		cd clients/python && uv sync; \
	else \
		echo "uv not found, skipping dependency installation"; \
	fi
	@echo "Python client generated successfully!"
	@echo "Review changes in clients/python/ before committing"

# Testing
test: ## Run Go tests
	@echo "Running Go tests..."
	@go test -v ./...

test-coverage: ## Run Go tests with coverage
	@echo "Running Go tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-python: ## Run Python client tests
	@echo "Running Python client tests..."
	@cd clients/python && if command -v uv >/dev/null 2>&1; then \
		uv run python -m pytest; \
	else \
		python -m pytest; \
	fi

# Python package management
python-build: ## Build Python package
	@echo "Building Python package..."
	@cd clients/python && uv build
	@echo "Python package built in clients/python/dist/"

python-publish: python-build ## Publish Python package to PyPI
	@echo "Publishing Python package to PyPI..."
	@cd clients/python && uv publish
	@echo "Package published successfully!"

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf clients/python/dist/
	@rm -rf clients/python/.openapi-generator/
	@rm -f coverage.out coverage.html
	@echo "Cleanup complete!"

# Development helpers
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./...
	@echo "Code formatted!"

lint: ## Run Go linter
	@echo "Running Go linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install it with:"; \
		echo "go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t dj-set-downloader .
	@echo "Docker image built: dj-set-downloader"

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	@docker run -p 8000:8000 dj-set-downloader

# Full development cycle
dev-setup: setup-dev swagger-docs python-client ## Complete development setup
	@echo "Development environment ready!"
	@echo "Run 'make help' to see available targets"
	@echo "Run 'make run' to start the application"

# Quick development iteration
dev-update: swagger-docs python-client test ## Update docs, client, and run tests
	@echo "Development update complete!" 