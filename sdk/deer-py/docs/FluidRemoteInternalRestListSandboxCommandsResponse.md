# FluidRemoteInternalRestListSandboxCommandsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[FluidRemoteInternalStoreCommand]**](FluidRemoteInternalStoreCommand.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_list_sandbox_commands_response import FluidRemoteInternalRestListSandboxCommandsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestListSandboxCommandsResponse from a JSON string
fluid_remote_internal_rest_list_sandbox_commands_response_instance = FluidRemoteInternalRestListSandboxCommandsResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestListSandboxCommandsResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_list_sandbox_commands_response_dict = fluid_remote_internal_rest_list_sandbox_commands_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestListSandboxCommandsResponse from a dict
fluid_remote_internal_rest_list_sandbox_commands_response_from_dict = FluidRemoteInternalRestListSandboxCommandsResponse.from_dict(fluid_remote_internal_rest_list_sandbox_commands_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


