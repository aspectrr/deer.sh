# FluidRemoteInternalStoreCommandExecRecord


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**redacted** | **Dict[str, str]** | placeholders for secrets redaction | [optional] 
**timeout** | [**TimeDuration**](TimeDuration.md) |  | [optional] 
**user** | **str** |  | [optional] 
**work_dir** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_command_exec_record import FluidRemoteInternalStoreCommandExecRecord

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreCommandExecRecord from a JSON string
fluid_remote_internal_store_command_exec_record_instance = FluidRemoteInternalStoreCommandExecRecord.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreCommandExecRecord.to_json())

# convert the object into a dict
fluid_remote_internal_store_command_exec_record_dict = fluid_remote_internal_store_command_exec_record_instance.to_dict()
# create an instance of FluidRemoteInternalStoreCommandExecRecord from a dict
fluid_remote_internal_store_command_exec_record_from_dict = FluidRemoteInternalStoreCommandExecRecord.from_dict(fluid_remote_internal_store_command_exec_record_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


