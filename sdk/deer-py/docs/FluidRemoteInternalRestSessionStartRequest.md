# FluidRemoteInternalRestSessionStartRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificate_id** | **str** |  | [optional] 
**source_ip** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_session_start_request import FluidRemoteInternalRestSessionStartRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSessionStartRequest from a JSON string
fluid_remote_internal_rest_session_start_request_instance = FluidRemoteInternalRestSessionStartRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSessionStartRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_session_start_request_dict = fluid_remote_internal_rest_session_start_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestSessionStartRequest from a dict
fluid_remote_internal_rest_session_start_request_from_dict = FluidRemoteInternalRestSessionStartRequest.from_dict(fluid_remote_internal_rest_session_start_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


