# ProgressTrackDetails


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**current_track** | **str** |  | [optional] 
**processed_tracks** | **int** |  | [optional] 
**total_tracks** | **int** |  | [optional] 
**track_number** | **int** |  | [optional] 

## Example

```python
from dj_set_downloader.models.progress_track_details import ProgressTrackDetails

# TODO update the JSON string below
json = "{}"
# create an instance of ProgressTrackDetails from a JSON string
progress_track_details_instance = ProgressTrackDetails.from_json(json)
# print the JSON string representation of the object
print(ProgressTrackDetails.to_json())

# convert the object into a dict
progress_track_details_dict = progress_track_details_instance.to_dict()
# create an instance of ProgressTrackDetails from a dict
progress_track_details_from_dict = ProgressTrackDetails.from_dict(progress_track_details_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


