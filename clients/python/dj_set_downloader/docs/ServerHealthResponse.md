# ServerHealthResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | [optional] 

## Example

```python
from dj_set_downloader.models.server_health_response import ServerHealthResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ServerHealthResponse from a JSON string
server_health_response_instance = ServerHealthResponse.from_json(json)
# print the JSON string representation of the object
print(ServerHealthResponse.to_json())

# convert the object into a dict
server_health_response_dict = server_health_response_instance.to_dict()
# create an instance of ServerHealthResponse from a dict
server_health_response_from_dict = ServerHealthResponse.from_dict(server_health_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


