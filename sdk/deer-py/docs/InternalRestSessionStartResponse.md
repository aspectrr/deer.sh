# InternalRestSessionStartResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_session_start_response import InternalRestSessionStartResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSessionStartResponse from a JSON string
internal_rest_session_start_response_instance = InternalRestSessionStartResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestSessionStartResponse.to_json())

# convert the object into a dict
internal_rest_session_start_response_dict = internal_rest_session_start_response_instance.to_dict()
# create an instance of InternalRestSessionStartResponse from a dict
internal_rest_session_start_response_from_dict = InternalRestSessionStartResponse.from_dict(internal_rest_session_start_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


