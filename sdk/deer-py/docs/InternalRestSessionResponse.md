# InternalRestSessionResponse


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
from fluid.models.internal_rest_session_response import InternalRestSessionResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSessionResponse from a JSON string
internal_rest_session_response_instance = InternalRestSessionResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestSessionResponse.to_json())

# convert the object into a dict
internal_rest_session_response_dict = internal_rest_session_response_instance.to_dict()
# create an instance of InternalRestSessionResponse from a dict
internal_rest_session_response_from_dict = InternalRestSessionResponse.from_dict(internal_rest_session_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


