# FluidRemoteInternalRestSessionEndResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**session_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_session_end_response import FluidRemoteInternalRestSessionEndResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSessionEndResponse from a JSON string
fluid_remote_internal_rest_session_end_response_instance = FluidRemoteInternalRestSessionEndResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSessionEndResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_session_end_response_dict = fluid_remote_internal_rest_session_end_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestSessionEndResponse from a dict
fluid_remote_internal_rest_session_end_response_from_dict = FluidRemoteInternalRestSessionEndResponse.from_dict(fluid_remote_internal_rest_session_end_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


