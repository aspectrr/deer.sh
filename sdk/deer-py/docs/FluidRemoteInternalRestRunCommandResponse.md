# FluidRemoteInternalRestRunCommandResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | [**FluidRemoteInternalStoreCommand**](FluidRemoteInternalStoreCommand.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_run_command_response import FluidRemoteInternalRestRunCommandResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestRunCommandResponse from a JSON string
fluid_remote_internal_rest_run_command_response_instance = FluidRemoteInternalRestRunCommandResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestRunCommandResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_run_command_response_dict = fluid_remote_internal_rest_run_command_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestRunCommandResponse from a dict
fluid_remote_internal_rest_run_command_response_from_dict = FluidRemoteInternalRestRunCommandResponse.from_dict(fluid_remote_internal_rest_run_command_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


