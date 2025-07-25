# Python Client Development Workflow

This document explains how to work with the Python client for the DJ Set Downloader API.

## Overview

The Python client is automatically generated from the OpenAPI specification of the Go API. The workflow separates client generation (development time) from publishing (release time).

## Development Flow

### 1. Making API Changes

When you modify the Go API code:

1. **Update API code** in `internal/server/`, `cmd/`, etc.
2. **Add/update Swagger annotations** in your Go code using `@Summary`, `@Description`, etc.
3. **Generate the Python client locally**:
   ```bash
   make python-client
   ```

This command will:
- Regenerate Swagger docs from Go annotations
- Generate the Python client from the updated OpenAPI spec
- Install dependencies with `uv sync`

### 2. Testing Changes

After generating the client:

1. **Review the changes** in `clients/python/`
2. **Test the client** locally if needed
3. **Commit both API changes and client updates** together

### 3. Pull Request Validation

When you create a PR that affects the API:

- The **Validate Python Client** workflow runs automatically
- It ensures the committed client matches the current API
- If out of sync, it will fail and tell you to run `make python-client`

### 4. Publishing

When your PR is merged to `main`:

- If `clients/python/` has changes, the **Publish Python Client** workflow runs
- It automatically builds and publishes the package to PyPI
- Creates a GitHub release with the built artifacts

## Available Commands

The project uses a Makefile for all development tasks. Run `make help` to see all available targets.

### Key Makefile Targets

- `make python-client` - Complete client regeneration (docs + client + deps)
- `make swagger-docs` - Generate only the Swagger documentation from Go annotations
- `make dev-setup` - Complete development environment setup
- `make dev-update` - Update docs, client, and run tests (quick iteration)
- `make python-build` - Build Python package for distribution
- `make python-publish` - Build and publish to PyPI
- `make test` - Run Go tests
- `make test-python` - Run Python client tests

## Configuration

### OpenAPI Generator Config
Located in `clients/openapi-generator-config.yaml`:
- `packageName`: Python package name
- `packageVersion`: Version (update this for releases)
- `projectName`: Project name for metadata

### Package Configuration
Located in `clients/python/pyproject.toml`:
- Package metadata and dependencies
- Build configuration for `uv`

## Manual Publishing

If you need to publish manually:

```bash
make python-publish  # Requires UV_PUBLISH_TOKEN environment variable
```

Or step by step:
```bash
make python-build    # Build package only
make python-publish  # Build and publish
```

## Requirements

Most dependencies can be installed automatically:

```bash
make install-deps  # Installs all required dependencies
make dev-setup    # Complete development environment setup
```

Manual requirements:
- **Go**: For running the API and generating docs
- **Node.js/npm**: For OpenAPI Generator
- **uv**: For Python package management (`curl -LsSf https://astral.sh/uv/install.sh | sh`)
- **PyPI token**: Set `UV_PUBLISH_TOKEN` environment variable or `PYPI_TOKEN` secret in GitHub 