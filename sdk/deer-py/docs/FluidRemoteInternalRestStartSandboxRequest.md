# FluidRemoteInternalRestStartSandboxRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**wait_for_ip** | **bool** | optional; default false | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_start_sandbox_request import FluidRemoteInternalRestStartSandboxRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestStartSandboxRequest from a JSON string
fluid_remote_internal_rest_start_sandbox_request_instance = FluidRemoteInternalRestStartSandboxRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestStartSandboxRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_start_sandbox_request_dict = fluid_remote_internal_rest_start_sandbox_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestStartSandboxRequest from a dict
fluid_remote_internal_rest_start_sandbox_request_from_dict = FluidRemoteInternalRestStartSandboxRequest.from_dict(fluid_remote_internal_rest_start_sandbox_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


