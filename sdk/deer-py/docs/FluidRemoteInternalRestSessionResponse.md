# FluidRemoteInternalRestSessionResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificate_id** | **str** |  | [optional] 
**duration_seconds** | **int** |  | [optional] 
**ended_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**source_ip** | **str** |  | [optional] 
**started_at** | **str** |  | [optional] 
**status** | **str** |  | [optional] 
**user_id** | **str** |  | [optional] 
**vm_id** | **str** |  | [optional] 
**vm_ip_address** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_session_response import FluidRemoteInternalRestSessionResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSessionResponse from a JSON string
fluid_remote_internal_rest_session_response_instance = FluidRemoteInternalRestSessionResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSessionResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_session_response_dict = fluid_remote_internal_rest_session_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestSessionResponse from a dict
fluid_remote_internal_rest_session_response_from_dict = FluidRemoteInternalRestSessionResponse.from_dict(fluid_remote_internal_rest_session_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


