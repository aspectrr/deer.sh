# FluidRemoteInternalRestRunCommandRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** | required | [optional] 
**env** | **Dict[str, str]** | optional | [optional] 
**private_key_path** | **str** | optional; if empty, uses managed credentials (requires SSH CA) | [optional] 
**timeout_sec** | **int** | optional; default from service config | [optional] 
**user** | **str** | optional; defaults to \&quot;sandbox\&quot; when using managed credentials | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_run_command_request import FluidRemoteInternalRestRunCommandRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestRunCommandRequest from a JSON string
fluid_remote_internal_rest_run_command_request_instance = FluidRemoteInternalRestRunCommandRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestRunCommandRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_run_command_request_dict = fluid_remote_internal_rest_run_command_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestRunCommandRequest from a dict
fluid_remote_internal_rest_run_command_request_from_dict = FluidRemoteInternalRestRunCommandRequest.from_dict(fluid_remote_internal_rest_run_command_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


