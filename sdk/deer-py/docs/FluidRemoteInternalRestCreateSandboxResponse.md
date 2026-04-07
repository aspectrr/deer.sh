# FluidRemoteInternalRestCreateSandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ip_address** | **str** | populated when auto_start and wait_for_ip are true | [optional] 
**sandbox** | [**FluidRemoteInternalStoreSandbox**](FluidRemoteInternalStoreSandbox.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_create_sandbox_response import FluidRemoteInternalRestCreateSandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestCreateSandboxResponse from a JSON string
fluid_remote_internal_rest_create_sandbox_response_instance = FluidRemoteInternalRestCreateSandboxResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestCreateSandboxResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_create_sandbox_response_dict = fluid_remote_internal_rest_create_sandbox_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestCreateSandboxResponse from a dict
fluid_remote_internal_rest_create_sandbox_response_from_dict = FluidRemoteInternalRestCreateSandboxResponse.from_dict(fluid_remote_internal_rest_create_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


