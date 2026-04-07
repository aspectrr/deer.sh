# FluidRemoteInternalRestVmInfo


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**disk_path** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**persistent** | **bool** |  | [optional] 
**state** | **str** |  | [optional] 
**uuid** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_vm_info import FluidRemoteInternalRestVmInfo

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestVmInfo from a JSON string
fluid_remote_internal_rest_vm_info_instance = FluidRemoteInternalRestVmInfo.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestVmInfo.to_json())

# convert the object into a dict
fluid_remote_internal_rest_vm_info_dict = fluid_remote_internal_rest_vm_info_instance.to_dict()
# create an instance of FluidRemoteInternalRestVmInfo from a dict
fluid_remote_internal_rest_vm_info_from_dict = FluidRemoteInternalRestVmInfo.from_dict(fluid_remote_internal_rest_vm_info_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


