# FluidRemoteInternalStoreDiff


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**diff_json** | [**FluidRemoteInternalStoreChangeDiff**](FluidRemoteInternalStoreChangeDiff.md) | JSON-encoded change diff | [optional] 
**from_snapshot** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**to_snapshot** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_diff import FluidRemoteInternalStoreDiff

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreDiff from a JSON string
fluid_remote_internal_store_diff_instance = FluidRemoteInternalStoreDiff.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreDiff.to_json())

# convert the object into a dict
fluid_remote_internal_store_diff_dict = fluid_remote_internal_store_diff_instance.to_dict()
# create an instance of FluidRemoteInternalStoreDiff from a dict
fluid_remote_internal_store_diff_from_dict = FluidRemoteInternalStoreDiff.from_dict(fluid_remote_internal_store_diff_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


