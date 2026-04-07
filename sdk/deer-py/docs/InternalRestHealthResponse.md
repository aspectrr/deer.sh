# InternalRestHealthResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_health_response import InternalRestHealthResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestHealthResponse from a JSON string
internal_rest_health_response_instance = InternalRestHealthResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestHealthResponse.to_json())

# convert the object into a dict
internal_rest_health_response_dict = internal_rest_health_response_instance.to_dict()
# create an instance of InternalRestHealthResponse from a dict
internal_rest_health_response_from_dict = InternalRestHealthResponse.from_dict(internal_rest_health_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


