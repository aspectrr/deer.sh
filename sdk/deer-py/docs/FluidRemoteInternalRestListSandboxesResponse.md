# FluidRemoteInternalRestListSandboxesResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sandboxes** | [**List[FluidRemoteInternalRestSandboxInfo]**](FluidRemoteInternalRestSandboxInfo.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_list_sandboxes_response import FluidRemoteInternalRestListSandboxesResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestListSandboxesResponse from a JSON string
fluid_remote_internal_rest_list_sandboxes_response_instance = FluidRemoteInternalRestListSandboxesResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestListSandboxesResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_list_sandboxes_response_dict = fluid_remote_internal_rest_list_sandboxes_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestListSandboxesResponse from a dict
fluid_remote_internal_rest_list_sandboxes_response_from_dict = FluidRemoteInternalRestListSandboxesResponse.from_dict(fluid_remote_internal_rest_list_sandboxes_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


