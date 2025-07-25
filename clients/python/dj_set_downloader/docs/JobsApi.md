# dj_set_downloader.JobsApi

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_jobs_get**](JobsApi.md#api_jobs_get) | **GET** /api/jobs | List all jobs
[**api_jobs_id_cancel_post**](JobsApi.md#api_jobs_id_cancel_post) | **POST** /api/jobs/{id}/cancel | Cancel a job
[**api_jobs_id_get**](JobsApi.md#api_jobs_id_get) | **GET** /api/jobs/{id} | Get job status


# **api_jobs_get**
> JobResponse api_jobs_get(page=page, page_size=page_size)

List all jobs

Retrieves a paginated list of all processing jobs

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.job_response import JobResponse
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
    api_instance = dj_set_downloader.JobsApi(api_client)
    page = 1 # int | Page number (optional) (default to 1)
    page_size = 10 # int | Number of jobs per page (max 100) (optional) (default to 10)

    try:
        # List all jobs
        api_response = api_instance.api_jobs_get(page=page, page_size=page_size)
        print("The response of JobsApi->api_jobs_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling JobsApi->api_jobs_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int**| Page number | [optional] [default to 1]
 **page_size** | **int**| Number of jobs per page (max 100) | [optional] [default to 10]

### Return type

[**JobResponse**](JobResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Jobs retrieved successfully |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_jobs_id_cancel_post**
> ServerCancelResponse api_jobs_id_cancel_post(id)

Cancel a job

Cancels a running or pending processing job by ID

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.server_cancel_response import ServerCancelResponse
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
    api_instance = dj_set_downloader.JobsApi(api_client)
    id = 'id_example' # str | Job ID

    try:
        # Cancel a job
        api_response = api_instance.api_jobs_id_cancel_post(id)
        print("The response of JobsApi->api_jobs_id_cancel_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling JobsApi->api_jobs_id_cancel_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Job ID | 

### Return type

[**ServerCancelResponse**](ServerCancelResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Job cancelled successfully |  -  |
**400** | Job cannot be cancelled (invalid state) |  -  |
**404** | Job not found |  -  |
**500** | Internal server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_jobs_id_get**
> JobStatus api_jobs_id_get(id)

Get job status

Retrieves the current status and progress of a processing job by ID

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.job_status import JobStatus
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
    api_instance = dj_set_downloader.JobsApi(api_client)
    id = 'id_example' # str | Job ID

    try:
        # Get job status
        api_response = api_instance.api_jobs_id_get(id)
        print("The response of JobsApi->api_jobs_id_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling JobsApi->api_jobs_id_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Job ID | 

### Return type

[**JobStatus**](JobStatus.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Job status retrieved successfully |  -  |
**404** | Job not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

