from http import HTTPStatus
from typing import Any, Optional, Union

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.job_request import JobRequest
from ...models.server_error_response import ServerErrorResponse
from ...models.server_message_response import ServerMessageResponse
from ...types import Response


def _get_kwargs(
    *,
    body: JobRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/process",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: Union[AuthenticatedClient, Client], response: httpx.Response
) -> Optional[Union[ServerErrorResponse, ServerMessageResponse]]:
    if response.status_code == 202:
        response_202 = ServerMessageResponse.from_dict(response.json())

        return response_202
    if response.status_code == 400:
        response_400 = ServerErrorResponse.from_dict(response.json())

        return response_400
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: Union[AuthenticatedClient, Client], response: httpx.Response
) -> Response[Union[ServerErrorResponse, ServerMessageResponse]]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: Union[AuthenticatedClient, Client],
    body: JobRequest,
) -> Response[Union[ServerErrorResponse, ServerMessageResponse]]:
    """Start processing a DJ set URL

     Submits a job that downloads and processes the given DJ set URL using the supplied tracklist.

    Args:
        body (JobRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Union[ServerErrorResponse, ServerMessageResponse]]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: Union[AuthenticatedClient, Client],
    body: JobRequest,
) -> Optional[Union[ServerErrorResponse, ServerMessageResponse]]:
    """Start processing a DJ set URL

     Submits a job that downloads and processes the given DJ set URL using the supplied tracklist.

    Args:
        body (JobRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Union[ServerErrorResponse, ServerMessageResponse]
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: Union[AuthenticatedClient, Client],
    body: JobRequest,
) -> Response[Union[ServerErrorResponse, ServerMessageResponse]]:
    """Start processing a DJ set URL

     Submits a job that downloads and processes the given DJ set URL using the supplied tracklist.

    Args:
        body (JobRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Union[ServerErrorResponse, ServerMessageResponse]]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: Union[AuthenticatedClient, Client],
    body: JobRequest,
) -> Optional[Union[ServerErrorResponse, ServerMessageResponse]]:
    """Start processing a DJ set URL

     Submits a job that downloads and processes the given DJ set URL using the supplied tracklist.

    Args:
        body (JobRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Union[ServerErrorResponse, ServerMessageResponse]
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
