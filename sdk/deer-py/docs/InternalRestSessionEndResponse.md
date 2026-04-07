# InternalRestSessionEndResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**session_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_session_end_response import InternalRestSessionEndResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSessionEndResponse from a JSON string
internal_rest_session_end_response_instance = InternalRestSessionEndResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestSessionEndResponse.to_json())

# convert the object into a dict
internal_rest_session_end_response_dict = internal_rest_session_end_response_instance.to_dict()
# create an instance of InternalRestSessionEndResponse from a dict
internal_rest_session_end_response_from_dict = InternalRestSessionEndResponse.from_dict(internal_rest_session_end_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


