#!/usr/bin/env python3
"""
Example usage of the DJ Set Downloader API client.

This example demonstrates how to use the generated client library
to interact with the DJ Set Downloader API.
"""

import asyncio
from typing import Optional
from dj_set_downloader_api_client import Client, AuthenticatedClient
from dj_set_downloader_api_client.api.jobs import get_api_jobs
from dj_set_downloader_api_client.api.utility import get_health
from dj_set_downloader_api_client.types import Response

# Example configuration
API_BASE_URL = "http://localhost:8000"  # Replace with your API base URL


def example_sync_usage():
    """Example of synchronous API usage."""
    print("=== Synchronous API Usage ===")
    
    # Create a client
    client = Client(base_url=API_BASE_URL)
    
    # Health check
    try:
        health_response = get_health.sync(client=client)
        print(f"Health check: {health_response}")
    except Exception as e:
        print(f"Health check failed: {e}")
    
    # List jobs
    try:
        jobs_response = get_api_jobs.sync(client=client)
        print(f"Jobs response: {jobs_response}")
    except Exception as e:
        print(f"Failed to fetch jobs: {e}")


async def example_async_usage():
    """Example of asynchronous API usage."""
    print("\n=== Asynchronous API Usage ===")
    
    # Create a client
    client = Client(base_url=API_BASE_URL)
    
    # Health check
    try:
        health_response = await get_health.asyncio(client=client)
        print(f"Health check: {health_response}")
    except Exception as e:
        print(f"Health check failed: {e}")
    
    # List jobs
    try:
        jobs_response = await get_api_jobs.asyncio(client=client)
        print(f"Jobs response: {jobs_response}")
    except Exception as e:
        print(f"Failed to fetch jobs: {e}")


def example_detailed_response():
    """Example of getting detailed responses with status codes."""
    print("\n=== Detailed Response Usage ===")
    
    client = Client(base_url=API_BASE_URL)
    
    # Get detailed response for health check
    try:
        response = get_health.sync_detailed(client=client)
        print(f"Status code: {response.status_code}")
        print(f"Headers: {response.headers}")
        print(f"Parsed data: {response.parsed}")
    except Exception as e:
        print(f"Health check failed: {e}")


def example_with_authentication():
    """Example of using authenticated client."""
    print("\n=== Authenticated Client Usage ===")
    
    # If your API requires authentication, use AuthenticatedClient
    # Replace "your-token-here" with your actual token
    client = AuthenticatedClient(
        base_url=API_BASE_URL,
        token="your-token-here"
    )
    
    # Use the authenticated client the same way as the regular client
    try:
        health_response = get_health.sync(client=client)
        print(f"Authenticated health check: {health_response}")
    except Exception as e:
        print(f"Authenticated health check failed: {e}")


def example_custom_httpx_client():
    """Example of using custom httpx client configuration."""
    print("\n=== Custom HTTPX Client Usage ===")
    
    # You can customize the underlying httpx client
    def log_request(request):
        print(f"Request: {request.method} {request.url}")
    
    def log_response(response):
        print(f"Response: {response.status_code}")
    
    client = Client(
        base_url=API_BASE_URL,
        httpx_args={
            "event_hooks": {
                "request": [log_request],
                "response": [log_response]
            }
        }
    )
    
    try:
        health_response = get_health.sync(client=client)
        print(f"Health check with logging: {health_response}")
    except Exception as e:
        print(f"Health check with logging failed: {e}")


def main():
    """Main function to run all examples."""
    print("DJ Set Downloader API Client Examples")
    print("=" * 50)
    
    # Run synchronous examples
    example_sync_usage()
    
    # Run asynchronous examples
    asyncio.run(example_async_usage())
    
    # Run detailed response example
    example_detailed_response()
    
    # Run authentication example
    example_with_authentication()
    
    # Run custom httpx client example
    example_custom_httpx_client()
    
    print("\n" + "=" * 50)
    print("Examples completed!")
    print("\nNote: Some examples may fail if the API server is not running.")
    print("To start the API server, run: docker compose up")


if __name__ == "__main__":
    main()