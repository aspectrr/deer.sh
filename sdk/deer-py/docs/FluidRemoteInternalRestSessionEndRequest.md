# FluidRemoteInternalRestSessionEndRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**reason** | **str** |  | [optional] 
**session_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_session_end_request import FluidRemoteInternalRestSessionEndRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSessionEndRequest from a JSON string
fluid_remote_internal_rest_session_end_request_instance = FluidRemoteInternalRestSessionEndRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSessionEndRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_session_end_request_dict = fluid_remote_internal_rest_session_end_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestSessionEndRequest from a dict
fluid_remote_internal_rest_session_end_request_from_dict = FluidRemoteInternalRestSessionEndRequest.from_dict(fluid_remote_internal_rest_session_end_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


