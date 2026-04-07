# FluidRemoteInternalRestGetSandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[FluidRemoteInternalStoreCommand]**](FluidRemoteInternalStoreCommand.md) |  | [optional] 
**sandbox** | [**FluidRemoteInternalStoreSandbox**](FluidRemoteInternalStoreSandbox.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_get_sandbox_response import FluidRemoteInternalRestGetSandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestGetSandboxResponse from a JSON string
fluid_remote_internal_rest_get_sandbox_response_instance = FluidRemoteInternalRestGetSandboxResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestGetSandboxResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_get_sandbox_response_dict = fluid_remote_internal_rest_get_sandbox_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestGetSandboxResponse from a dict
fluid_remote_internal_rest_get_sandbox_response_from_dict = FluidRemoteInternalRestGetSandboxResponse.from_dict(fluid_remote_internal_rest_get_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


