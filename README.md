# DJ Set Downloader

This application downloads DJ sets from SoundCloud and splits them into individual tracks based on a tracklist.

## Testing

To run all tests:

```bash
go test ./...
```

To generate test coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Categories

- **Unit Tests**: Test individual functions and components in isolation
- **Integration Tests**: Test components that interact with external systems (these are skipped by default)

## Continuous Integration

This project uses GitHub Actions for continuous integration. The following checks run automatically on each push to the main branch:

1. **Linting**: Using golangci-lint to ensure code quality
2. **Testing**: Running all unit tests and generating coverage reports
3. **Building**: Ensuring the application builds successfully

### CI Workflow Files

- `.github/workflows/ci.yml`: Main CI workflow for linting, testing, and building
- `.github/workflows/go-test.yml`: Specialized workflow for running tests with coverage reports

## Development Setup

To set up for local development:

1. Install Go (version 1.24.0 or later recommended)
2. Install FFMPEG for audio processing
3. Install Python and the `scdl` package for SoundCloud downloading
4. Clone the repository
5. Run `go mod download` to fetch dependencies
6. Run tests to verify your setup

## Configuration

The application is configured via the `config/config.yaml` file. See the example config file for available options.

## Dependencies

* [ffmpeg](https://github.com/FFmpeg/FFmpeg)
* [scdl](https://github.com/scdl-org/scdl)

## Usage

`go run cmd/main.go -tracklist-url https://www.1001tracklists.com/tracklist/nn39729/mind-against-afterlife-voyage-006-2017-10-18.html`

Original DJ Set on SoundCloud:

[![3HmoNTX.md.png](https://iili.io/3HmoNTX.md.png)](https://freeimage.host/i/3HmoNTX)

After Processing with DJ Set Downloader – Individual Tracks in Apple Music:

[![3HmoXYN.md.png](https://iili.io/3HmoXYN.md.png)](https://freeimage.host/i/3HmoXYN)