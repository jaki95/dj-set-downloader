# dj_set_downloader.DownloadsApi

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**api_jobs_id_download_get**](DownloadsApi.md#api_jobs_id_download_get) | **GET** /api/jobs/{id}/download | Download all tracks as ZIP
[**api_jobs_id_tracks_get**](DownloadsApi.md#api_jobs_id_tracks_get) | **GET** /api/jobs/{id}/tracks | Get tracks information
[**api_jobs_id_tracks_track_number_download_get**](DownloadsApi.md#api_jobs_id_tracks_track_number_download_get) | **GET** /api/jobs/{id}/tracks/{trackNumber}/download | Download a single track


# **api_jobs_id_download_get**
> bytearray api_jobs_id_download_get(id)

Download all tracks as ZIP

Downloads all processed tracks for a completed job as a ZIP archive

### Example


```python
import dj_set_downloader
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
    api_instance = dj_set_downloader.DownloadsApi(api_client)
    id = 'id_example' # str | Job ID

    try:
        # Download all tracks as ZIP
        api_response = api_instance.api_jobs_id_download_get(id)
        print("The response of DownloadsApi->api_jobs_id_download_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DownloadsApi->api_jobs_id_download_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Job ID | 

### Return type

**bytearray**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/zip

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Job is not completed yet |  -  |
**404** | Job not found or no tracks available |  -  |
**500** | Server error during ZIP creation |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_jobs_id_tracks_get**
> JobTracksInfoResponse api_jobs_id_tracks_get(id)

Get tracks information

Retrieves metadata and download information for all tracks in a completed job

### Example


```python
import dj_set_downloader
from dj_set_downloader.models.job_tracks_info_response import JobTracksInfoResponse
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
    api_instance = dj_set_downloader.DownloadsApi(api_client)
    id = 'id_example' # str | Job ID

    try:
        # Get tracks information
        api_response = api_instance.api_jobs_id_tracks_get(id)
        print("The response of DownloadsApi->api_jobs_id_tracks_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DownloadsApi->api_jobs_id_tracks_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Job ID | 

### Return type

[**JobTracksInfoResponse**](JobTracksInfoResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Tracks information retrieved successfully |  -  |
**400** | Job is not completed yet |  -  |
**404** | Job not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **api_jobs_id_tracks_track_number_download_get**
> bytearray api_jobs_id_tracks_track_number_download_get(id, track_number)

Download a single track

Downloads a specific processed track by job ID and track number

### Example


```python
import dj_set_downloader
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
    api_instance = dj_set_downloader.DownloadsApi(api_client)
    id = 'id_example' # str | Job ID
    track_number = 56 # int | Track number (1-based)

    try:
        # Download a single track
        api_response = api_instance.api_jobs_id_tracks_track_number_download_get(id, track_number)
        print("The response of DownloadsApi->api_jobs_id_tracks_track_number_download_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DownloadsApi->api_jobs_id_tracks_track_number_download_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Job ID | 
 **track_number** | **int**| Track number (1-based) | 

### Return type

**bytearray**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: audio/mpeg, audio/flac, audio/wav

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Invalid track number or job not completed |  -  |
**404** | Job or track not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

