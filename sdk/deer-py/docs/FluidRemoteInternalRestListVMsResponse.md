# FluidRemoteInternalRestListVMsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**vms** | [**List[FluidRemoteInternalRestVmInfo]**](FluidRemoteInternalRestVmInfo.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_list_vms_response import FluidRemoteInternalRestListVMsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestListVMsResponse from a JSON string
fluid_remote_internal_rest_list_vms_response_instance = FluidRemoteInternalRestListVMsResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestListVMsResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_list_vms_response_dict = fluid_remote_internal_rest_list_vms_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestListVMsResponse from a dict
fluid_remote_internal_rest_list_vms_response_from_dict = FluidRemoteInternalRestListVMsResponse.from_dict(fluid_remote_internal_rest_list_vms_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


