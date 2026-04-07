# InternalRestListSessionsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sessions** | [**List[InternalRestSessionResponse]**](InternalRestSessionResponse.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_list_sessions_response import InternalRestListSessionsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestListSessionsResponse from a JSON string
internal_rest_list_sessions_response_instance = InternalRestListSessionsResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestListSessionsResponse.to_json())

# convert the object into a dict
internal_rest_list_sessions_response_dict = internal_rest_list_sessions_response_instance.to_dict()
# create an instance of InternalRestListSessionsResponse from a dict
internal_rest_list_sessions_response_from_dict = InternalRestListSessionsResponse.from_dict(internal_rest_list_sessions_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


