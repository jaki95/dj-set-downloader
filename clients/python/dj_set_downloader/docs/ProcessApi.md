# dj_set_downloader.ProcessApi

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_process_post**](ProcessApi.md#api_process_post) | **POST** /api/process | Process a DJ set URL with tracklist


# **api_process_post**
> ServerProcessResponse api_process_post(request)

Process a DJ set URL with tracklist

Starts processing a DJ set from a given URL using the provided tracklist. Returns a job ID for tracking progress.

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.job_request import JobRequest
from dj_set_downloader.models.server_process_response import ServerProcessResponse
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
    api_instance = dj_set_downloader.ProcessApi(api_client)
    request = dj_set_downloader.JobRequest() # JobRequest | Processing request with URL and tracklist

    try:
        # Process a DJ set URL with tracklist
        api_response = api_instance.api_process_post(request)
        print("The response of ProcessApi->api_process_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ProcessApi->api_process_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**JobRequest**](JobRequest.md)| Processing request with URL and tracklist | 

### Return type

[**ServerProcessResponse**](ServerProcessResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**202** | Processing started successfully |  -  |
**400** | Invalid request or tracklist |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

