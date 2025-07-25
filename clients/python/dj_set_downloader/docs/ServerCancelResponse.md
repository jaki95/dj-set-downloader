# ServerCancelResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 

## Example

```python
from dj_set_downloader.models.server_cancel_response import ServerCancelResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ServerCancelResponse from a JSON string
server_cancel_response_instance = ServerCancelResponse.from_json(json)
# print the JSON string representation of the object
print(ServerCancelResponse.to_json())

# convert the object into a dict
server_cancel_response_dict = server_cancel_response_instance.to_dict()
# create an instance of ServerCancelResponse from a dict
server_cancel_response_from_dict = ServerCancelResponse.from_dict(server_cancel_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


