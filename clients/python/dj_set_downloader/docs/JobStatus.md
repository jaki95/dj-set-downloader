# JobStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**download_all_url** | **str** | Additional fields from main branch | [optional] 
**end_time** | **str** |  | [optional] 
**error** | **str** |  | [optional] 
**events** | [**List[ProgressEvent]**](ProgressEvent.md) |  | [optional] 
**id** | **str** |  | [optional] 
**message** | **str** |  | [optional] 
**progress** | **float** |  | [optional] 
**results** | **List[str]** |  | [optional] 
**start_time** | **str** |  | [optional] 
**status** | **str** |  | [optional] 
**total_tracks** | **int** |  | [optional] 
**tracklist** | [**DomainTracklist**](DomainTracklist.md) |  | [optional] 

## Example

```python
from dj_set_downloader.models.job_status import JobStatus

# TODO update the JSON string below
json = "{}"
# create an instance of JobStatus from a JSON string
job_status_instance = JobStatus.from_json(json)
# print the JSON string representation of the object
print(JobStatus.to_json())

# convert the object into a dict
job_status_dict = job_status_instance.to_dict()
# create an instance of JobStatus from a dict
job_status_from_dict = JobStatus.from_dict(job_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


