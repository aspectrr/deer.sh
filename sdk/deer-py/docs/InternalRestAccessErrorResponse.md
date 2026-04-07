# InternalRestAccessErrorResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **int** |  | [optional] 
**details** | **str** |  | [optional] 
**error** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_access_error_response import InternalRestAccessErrorResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestAccessErrorResponse from a JSON string
internal_rest_access_error_response_instance = InternalRestAccessErrorResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestAccessErrorResponse.to_json())

# convert the object into a dict
internal_rest_access_error_response_dict = internal_rest_access_error_response_instance.to_dict()
# create an instance of InternalRestAccessErrorResponse from a dict
internal_rest_access_error_response_from_dict = InternalRestAccessErrorResponse.from_dict(internal_rest_access_error_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


