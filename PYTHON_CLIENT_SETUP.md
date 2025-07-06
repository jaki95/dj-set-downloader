# DJ Set Downloader Python Client Library

This document provides instructions for setting up and using the Python client library for the DJ Set Downloader API.

## Overview

The DJ Set Downloader Python client library has been successfully generated from the OpenAPI specification. It provides a type-safe, modern Python interface for interacting with the DJ Set Downloader API.

## Generated Structure

```
dj_set_downloader_client/
├── dj_set_downloader_api_client/     # Main client package
│   ├── __init__.py                   # Package initialization
│   ├── client.py                     # Client classes
│   ├── errors.py                     # Error definitions
│   ├── types.py                      # Response types
│   ├── api/                          # API endpoint functions
│   │   ├── jobs/                     # Job-related endpoints
│   │   │   ├── get_api_jobs.py       # List jobs
│   │   │   └── post_api_process.py   # Process DJ sets
│   │   └── utility/                  # Utility endpoints
│   │       └── get_health.py         # Health check
│   └── models/                       # Data models
│       ├── job_request.py            # Job request model
│       ├── job_response.py           # Job response model
│       ├── job_status.py             # Job status model
│       ├── domain_track.py           # Track model
│       ├── domain_tracklist.py       # Tracklist model
│       ├── progress_event.py         # Progress event model
│       ├── progress_stage.py         # Progress stage enum
│       ├── progress_track_details.py # Track details model
│       ├── server_error_response.py  # Error response model
│       └── server_message_response.py # Message response model
├── example_usage.py                  # Usage examples
├── pyproject.toml                    # Poetry configuration
├── README.md                         # Generated README
└── CHANGELOG.md                      # Version history
```

## Installation

### Option 1: Using Poetry (Recommended)

1. **Install Poetry** (if not already installed):
   ```bash
   curl -sSL https://install.python-poetry.org | python3 -
   ```

2. **Install the client library**:
   ```bash
   cd dj_set_downloader_client
   poetry install
   ```

3. **Activate the virtual environment**:
   ```bash
   poetry shell
   ```

### Option 2: Using pip

1. **Build the wheel**:
   ```bash
   cd dj_set_downloader_client
   pip install poetry
   poetry build -f wheel
   ```

2. **Install the wheel**:
   ```bash
   pip install dist/*.whl
   ```

### Option 3: Development Installation

For development purposes, you can install the client in editable mode:

```bash
cd dj_set_downloader_client
pip install -e .
```

## Basic Usage

### Creating a Client

```python
from dj_set_downloader_api_client import Client

# Create a basic client
client = Client(base_url="http://localhost:8000")

# For APIs requiring authentication
from dj_set_downloader_api_client import AuthenticatedClient

auth_client = AuthenticatedClient(
    base_url="http://localhost:8000",
    token="your-api-token"
)
```

### Making API Calls

```python
from dj_set_downloader_api_client.api.utility import get_health
from dj_set_downloader_api_client.api.jobs import get_api_jobs

# Health check
health = get_health.sync(client=client)
print(f"Health: {health}")

# List jobs
jobs = get_api_jobs.sync(client=client)
print(f"Jobs: {jobs}")
```

### Async Usage

```python
import asyncio

async def main():
    # Async health check
    health = await get_health.asyncio(client=client)
    print(f"Health: {health}")
    
    # Async job listing
    jobs = await get_api_jobs.asyncio(client=client)
    print(f"Jobs: {jobs}")

# Run the async function
asyncio.run(main())
```

### Processing DJ Sets

```python
from dj_set_downloader_api_client.api.jobs import post_api_process
from dj_set_downloader_api_client.models import JobRequest
import json

# Create a job request
tracklist = [
    {"artist": "Artist 1", "name": "Track 1", "start_time": "00:00"},
    {"artist": "Artist 2", "name": "Track 2", "start_time": "03:45"}
]

job_request = JobRequest(
    url="https://soundcloud.com/example/set",
    tracklist=json.dumps(tracklist),
    file_extension="mp3",
    max_concurrent_tasks=3
)

# Submit the job
result = post_api_process.sync(client=client, body=job_request)
print(f"Job submitted: {result}")
```

## Client Configuration

### Timeout Settings

```python
import httpx

client = Client(
    base_url="http://localhost:8000",
    timeout=httpx.Timeout(30.0)  # 30 seconds timeout
)
```

### SSL Configuration

```python
# Disable SSL verification (for development only)
client = Client(
    base_url="https://localhost:8000",
    verify_ssl=False
)

# Use custom certificate
client = Client(
    base_url="https://localhost:8000",
    verify_ssl="/path/to/cert.pem"
)
```

### Custom Headers and Cookies

```python
client = Client(
    base_url="http://localhost:8000",
    headers={"Custom-Header": "value"},
    cookies={"session": "abc123"}
)
```

### Advanced httpx Configuration

```python
def log_request(request):
    print(f"Request: {request.method} {request.url}")

def log_response(response):
    print(f"Response: {response.status_code}")

client = Client(
    base_url="http://localhost:8000",
    httpx_args={
        "event_hooks": {
            "request": [log_request],
            "response": [log_response]
        }
    }
)
```

## Error Handling

```python
from dj_set_downloader_api_client.errors import UnexpectedStatus
import httpx

try:
    result = get_health.sync(client=client)
except UnexpectedStatus as e:
    print(f"Unexpected status: {e.status_code}")
    print(f"Response content: {e.content}")
except httpx.TimeoutException:
    print("Request timed out")
except Exception as e:
    print(f"Error: {e}")
```

## Testing

Run the example usage:

```bash
cd dj_set_downloader_client
python example_usage.py
```

Note: Make sure the DJ Set Downloader API server is running on `http://localhost:8000` before running the examples.

## Building and Distribution

### Build the Package

```bash
cd dj_set_downloader_client
poetry build
```

This creates both wheel and source distributions in the `dist/` directory.

### Publish to PyPI

```bash
# Configure PyPI credentials (one-time setup)
poetry config pypi-token.pypi your-pypi-token

# Publish to PyPI
poetry publish --build
```

### Publish to Private Repository

```bash
# Configure private repository
poetry config repositories.private-repo https://your-repo.com/simple/
poetry config http-basic.private-repo username password

# Publish to private repository
poetry publish --build -r private-repo
```

## API Reference

### Available Endpoints

- **Health Check**: `GET /health`
- **List Jobs**: `GET /api/jobs`
- **Process DJ Set**: `POST /api/process`

### Data Models

All API responses are automatically parsed into Python objects with proper type hints:

- `JobRequest`: Request to process a DJ set
- `JobResponse`: Response containing job list
- `JobStatus`: Individual job status
- `DomainTrack`: Track information
- `DomainTracklist`: Tracklist information
- `ProgressEvent`: Progress update event
- `ProgressStage`: Progress stage enumeration
- `ServerErrorResponse`: Error response
- `ServerMessageResponse`: Success message response

## Troubleshooting

### Import Errors

If you encounter import errors, ensure the client is properly installed:

```bash
pip show dj-set-downloader-api-client
```

### Connection Issues

Verify the API server is running:

```bash
curl http://localhost:8000/health
```

### SSL Certificate Issues

For development, you can disable SSL verification:

```python
client = Client(base_url="https://localhost:8000", verify_ssl=False)
```

## Contributing

To contribute to the client library:

1. Make changes to the OpenAPI specification in the main project
2. Regenerate the client using the provided scripts
3. Update examples and documentation as needed
4. Test the changes thoroughly

## License

This client library is generated from the DJ Set Downloader API and inherits its license terms.