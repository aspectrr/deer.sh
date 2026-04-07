# InternalRestListVMsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**host_errors** | [**List[InternalRestHostError]**](InternalRestHostError.md) | Errors from unreachable hosts (multi-host mode) | [optional] 
**vms** | [**List[InternalRestVmInfo]**](InternalRestVmInfo.md) |  | [optional] 

## Example

```python
from fluid.models.internal_rest_list_vms_response import InternalRestListVMsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestListVMsResponse from a JSON string
internal_rest_list_vms_response_instance = InternalRestListVMsResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestListVMsResponse.to_json())

# convert the object into a dict
internal_rest_list_vms_response_dict = internal_rest_list_vms_response_instance.to_dict()
# create an instance of InternalRestListVMsResponse from a dict
internal_rest_list_vms_response_from_dict = InternalRestListVMsResponse.from_dict(internal_rest_list_vms_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


