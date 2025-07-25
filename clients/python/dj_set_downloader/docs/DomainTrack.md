# DomainTrack


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artist** | **str** |  | [optional] 
**available** | **bool** |  | [optional] 
**download_url** | **str** |  | [optional] 
**end_time** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**size_bytes** | **int** |  | [optional] 
**start_time** | **str** |  | [optional] 
**track_number** | **int** |  | [optional] 

## Example

```python
from dj_set_downloader.models.domain_track import DomainTrack

# TODO update the JSON string below
json = "{}"
# create an instance of DomainTrack from a JSON string
domain_track_instance = DomainTrack.from_json(json)
# print the JSON string representation of the object
print(DomainTrack.to_json())

# convert the object into a dict
domain_track_dict = domain_track_instance.to_dict()
# create an instance of DomainTrack from a dict
domain_track_from_dict = DomainTrack.from_dict(domain_track_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


