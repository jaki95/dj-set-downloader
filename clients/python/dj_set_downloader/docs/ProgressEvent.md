# ProgressEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**data** | **List[int]** |  | [optional] 
**error** | **str** |  | [optional] 
**message** | **str** |  | [optional] 
**progress** | **float** |  | [optional] 
**stage** | [**ProgressStage**](ProgressStage.md) |  | [optional] 
**timestamp** | **str** |  | [optional] 
**track_details** | [**ProgressTrackDetails**](ProgressTrackDetails.md) |  | [optional] 

## Example

```python
from dj_set_downloader.models.progress_event import ProgressEvent

# TODO update the JSON string below
json = "{}"
# create an instance of ProgressEvent from a JSON string
progress_event_instance = ProgressEvent.from_json(json)
# print the JSON string representation of the object
print(ProgressEvent.to_json())

# convert the object into a dict
progress_event_dict = progress_event_instance.to_dict()
# create an instance of ProgressEvent from a dict
progress_event_from_dict = ProgressEvent.from_dict(progress_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


