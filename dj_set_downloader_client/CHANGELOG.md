# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-06

### Added
- Initial release of the DJ Set Downloader API client
- Support for all API endpoints:
  - Health check (`/health`)
  - List jobs (`/api/jobs`)
  - Process DJ sets (`/api/process`)
- Both synchronous and asynchronous client support
- Comprehensive type hints and data models
- Support for authentication
- Configurable httpx client settings
- Example usage scripts

### Features
- **Client Types**: Both `Client` and `AuthenticatedClient` for different use cases
- **Async Support**: Full async/await support with `asyncio` methods
- **Type Safety**: Complete type hints for all models and responses
- **Detailed Responses**: Access to response status codes, headers, and raw content
- **Customizable**: Configurable timeouts, SSL verification, and httpx settings
- **Modern**: Built with modern Python (3.9+) and httpx