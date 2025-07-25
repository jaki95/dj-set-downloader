# DomainTracklist


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artist** | **str** |  | [optional] 
**genre** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**tracks** | [**List[DomainTrack]**](DomainTrack.md) |  | [optional] 
**year** | **int** |  | [optional] 

## Example

```python
from dj_set_downloader.models.domain_tracklist import DomainTracklist

# TODO update the JSON string below
json = "{}"
# create an instance of DomainTracklist from a JSON string
domain_tracklist_instance = DomainTracklist.from_json(json)
# print the JSON string representation of the object
print(DomainTracklist.to_json())

# convert the object into a dict
domain_tracklist_dict = domain_tracklist_instance.to_dict()
# create an instance of DomainTracklist from a dict
domain_tracklist_from_dict = DomainTracklist.from_dict(domain_tracklist_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


