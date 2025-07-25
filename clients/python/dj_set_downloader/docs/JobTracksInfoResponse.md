# JobTracksInfoResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**download_all_url** | **str** |  | [optional] 
**job_id** | **str** |  | [optional] 
**total_tracks** | **int** |  | [optional] 
**tracks** | [**List[DomainTrack]**](DomainTrack.md) |  | [optional] 

## Example

```python
from dj_set_downloader.models.job_tracks_info_response import JobTracksInfoResponse

# TODO update the JSON string below
json = "{}"
# create an instance of JobTracksInfoResponse from a JSON string
job_tracks_info_response_instance = JobTracksInfoResponse.from_json(json)
# print the JSON string representation of the object
print(JobTracksInfoResponse.to_json())

# convert the object into a dict
job_tracks_info_response_dict = job_tracks_info_response_instance.to_dict()
# create an instance of JobTracksInfoResponse from a dict
job_tracks_info_response_from_dict = JobTracksInfoResponse.from_dict(job_tracks_info_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


