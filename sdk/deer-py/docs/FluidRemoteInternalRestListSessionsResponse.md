# FluidRemoteInternalRestListSessionsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sessions** | [**List[FluidRemoteInternalRestSessionResponse]**](FluidRemoteInternalRestSessionResponse.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_list_sessions_response import FluidRemoteInternalRestListSessionsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestListSessionsResponse from a JSON string
fluid_remote_internal_rest_list_sessions_response_instance = FluidRemoteInternalRestListSessionsResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestListSessionsResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_list_sessions_response_dict = fluid_remote_internal_rest_list_sessions_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestListSessionsResponse from a dict
fluid_remote_internal_rest_list_sessions_response_from_dict = FluidRemoteInternalRestListSessionsResponse.from_dict(fluid_remote_internal_rest_list_sessions_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


