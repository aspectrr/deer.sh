# InternalRestListSandboxCommandsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStoreCommand]**](GithubComAspectrrFluidShFluidRemoteInternalStoreCommand.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_list_sandbox_commands_response import InternalRestListSandboxCommandsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestListSandboxCommandsResponse from a JSON string
internal_rest_list_sandbox_commands_response_instance = InternalRestListSandboxCommandsResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestListSandboxCommandsResponse.to_json())

# convert the object into a dict
internal_rest_list_sandbox_commands_response_dict = internal_rest_list_sandbox_commands_response_instance.to_dict()
# create an instance of InternalRestListSandboxCommandsResponse from a dict
internal_rest_list_sandbox_commands_response_from_dict = InternalRestListSandboxCommandsResponse.from_dict(internal_rest_list_sandbox_commands_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


