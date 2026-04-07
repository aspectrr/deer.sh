# FluidRemoteInternalErrorErrorResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **int** |  | [optional] 
**details** | **str** |  | [optional] 
**error** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_error_error_response import FluidRemoteInternalErrorErrorResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalErrorErrorResponse from a JSON string
fluid_remote_internal_error_error_response_instance = FluidRemoteInternalErrorErrorResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalErrorErrorResponse.to_json())

# convert the object into a dict
fluid_remote_internal_error_error_response_dict = fluid_remote_internal_error_error_response_instance.to_dict()
# create an instance of FluidRemoteInternalErrorErrorResponse from a dict
fluid_remote_internal_error_error_response_from_dict = FluidRemoteInternalErrorErrorResponse.from_dict(fluid_remote_internal_error_error_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


