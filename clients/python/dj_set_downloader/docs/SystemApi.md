# dj_set_downloader.SystemApi

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**health_get**](SystemApi.md#health_get) | **GET** /health | Health check


# **health_get**
> ServerHealthResponse health_get()

Health check

Returns the health status of the API

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.server_health_response import ServerHealthResponse
from dj_set_downloader.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8000
# See configuration.py for a list of all supported configuration parameters.
configuration = dj_set_downloader.Configuration(
    host = "http://localhost:8000"
)


# Enter a context with an instance of the API client
with dj_set_downloader.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = dj_set_downloader.SystemApi(api_client)

    try:
        # Health check
        api_response = api_instance.health_get()
        print("The response of SystemApi->health_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SystemApi->health_get: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**ServerHealthResponse**](ServerHealthResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Service is healthy |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

