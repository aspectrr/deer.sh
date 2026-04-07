# InternalRestSessionEndRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**reason** | **str** |  | [optional] 
**session_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_session_end_request import InternalRestSessionEndRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSessionEndRequest from a JSON string
internal_rest_session_end_request_instance = InternalRestSessionEndRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestSessionEndRequest.to_json())

# convert the object into a dict
internal_rest_session_end_request_dict = internal_rest_session_end_request_instance.to_dict()
# create an instance of InternalRestSessionEndRequest from a dict
internal_rest_session_end_request_from_dict = InternalRestSessionEndRequest.from_dict(internal_rest_session_end_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


