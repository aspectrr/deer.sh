# InternalRestSessionStartRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificate_id** | **str** |  | [optional] 
**source_ip** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_session_start_request import InternalRestSessionStartRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSessionStartRequest from a JSON string
internal_rest_session_start_request_instance = InternalRestSessionStartRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestSessionStartRequest.to_json())

# convert the object into a dict
internal_rest_session_start_request_dict = internal_rest_session_start_request_instance.to_dict()
# create an instance of InternalRestSessionStartRequest from a dict
internal_rest_session_start_request_from_dict = InternalRestSessionStartRequest.from_dict(internal_rest_session_start_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


