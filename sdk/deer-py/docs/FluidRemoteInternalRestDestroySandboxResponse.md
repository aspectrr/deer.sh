# FluidRemoteInternalRestDestroySandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**base_image** | **str** |  | [optional] 
**sandbox_name** | **str** |  | [optional] 
**state** | [**FluidRemoteInternalStoreSandboxState**](FluidRemoteInternalStoreSandboxState.md) |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_destroy_sandbox_response import FluidRemoteInternalRestDestroySandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestDestroySandboxResponse from a JSON string
fluid_remote_internal_rest_destroy_sandbox_response_instance = FluidRemoteInternalRestDestroySandboxResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestDestroySandboxResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_destroy_sandbox_response_dict = fluid_remote_internal_rest_destroy_sandbox_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestDestroySandboxResponse from a dict
fluid_remote_internal_rest_destroy_sandbox_response_from_dict = FluidRemoteInternalRestDestroySandboxResponse.from_dict(fluid_remote_internal_rest_destroy_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


