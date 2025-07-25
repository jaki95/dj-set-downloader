# ServerProcessResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_id** | **str** |  | [optional] 
**message** | **str** |  | [optional] 

## Example

```python
from dj_set_downloader.models.server_process_response import ServerProcessResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ServerProcessResponse from a JSON string
server_process_response_instance = ServerProcessResponse.from_json(json)
# print the JSON string representation of the object
print(ServerProcessResponse.to_json())

# convert the object into a dict
server_process_response_dict = server_process_response_instance.to_dict()
# create an instance of ServerProcessResponse from a dict
server_process_response_from_dict = ServerProcessResponse.from_dict(server_process_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


