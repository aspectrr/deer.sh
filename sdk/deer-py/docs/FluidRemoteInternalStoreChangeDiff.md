# FluidRemoteInternalStoreChangeDiff


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands_run** | [**List[FluidRemoteInternalStoreCommandSummary]**](FluidRemoteInternalStoreCommandSummary.md) |  | [optional] 
**files_added** | **List[str]** |  | [optional] 
**files_modified** | **List[str]** |  | [optional] 
**files_removed** | **List[str]** |  | [optional] 
**packages_added** | [**List[FluidRemoteInternalStorePackageInfo]**](FluidRemoteInternalStorePackageInfo.md) |  | [optional] 
**packages_removed** | [**List[FluidRemoteInternalStorePackageInfo]**](FluidRemoteInternalStorePackageInfo.md) |  | [optional] 
**services_changed** | [**List[FluidRemoteInternalStoreServiceChange]**](FluidRemoteInternalStoreServiceChange.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_change_diff import FluidRemoteInternalStoreChangeDiff

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreChangeDiff from a JSON string
fluid_remote_internal_store_change_diff_instance = FluidRemoteInternalStoreChangeDiff.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreChangeDiff.to_json())

# convert the object into a dict
fluid_remote_internal_store_change_diff_dict = fluid_remote_internal_store_change_diff_instance.to_dict()
# create an instance of FluidRemoteInternalStoreChangeDiff from a dict
fluid_remote_internal_store_change_diff_from_dict = FluidRemoteInternalStoreChangeDiff.from_dict(fluid_remote_internal_store_change_diff_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


