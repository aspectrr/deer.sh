# FluidRemoteInternalStoreCommand


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** |  | [optional] 
**ended_at** | **str** |  | [optional] 
**env_json** | **str** | JSON-encoded env map | [optional] 
**exit_code** | **int** |  | [optional] 
**id** | **str** |  | [optional] 
**metadata** | [**FluidRemoteInternalStoreCommandExecRecord**](FluidRemoteInternalStoreCommandExecRecord.md) |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**started_at** | **str** |  | [optional] 
**stderr** | **str** |  | [optional] 
**stdout** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_command import FluidRemoteInternalStoreCommand

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreCommand from a JSON string
fluid_remote_internal_store_command_instance = FluidRemoteInternalStoreCommand.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreCommand.to_json())

# convert the object into a dict
fluid_remote_internal_store_command_dict = fluid_remote_internal_store_command_instance.to_dict()
# create an instance of FluidRemoteInternalStoreCommand from a dict
fluid_remote_internal_store_command_from_dict = FluidRemoteInternalStoreCommand.from_dict(fluid_remote_internal_store_command_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


