# FluidRemoteInternalStoreCommandSummary


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**at** | **str** |  | [optional] 
**cmd** | **str** |  | [optional] 
**exit_code** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_command_summary import FluidRemoteInternalStoreCommandSummary

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreCommandSummary from a JSON string
fluid_remote_internal_store_command_summary_instance = FluidRemoteInternalStoreCommandSummary.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreCommandSummary.to_json())

# convert the object into a dict
fluid_remote_internal_store_command_summary_dict = fluid_remote_internal_store_command_summary_instance.to_dict()
# create an instance of FluidRemoteInternalStoreCommandSummary from a dict
fluid_remote_internal_store_command_summary_from_dict = FluidRemoteInternalStoreCommandSummary.from_dict(fluid_remote_internal_store_command_summary_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


