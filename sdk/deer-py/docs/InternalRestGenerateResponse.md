# InternalRestGenerateResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**note** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_generate_response import InternalRestGenerateResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestGenerateResponse from a JSON string
internal_rest_generate_response_instance = InternalRestGenerateResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestGenerateResponse.to_json())

# convert the object into a dict
internal_rest_generate_response_dict = internal_rest_generate_response_instance.to_dict()
# create an instance of InternalRestGenerateResponse from a dict
internal_rest_generate_response_from_dict = InternalRestGenerateResponse.from_dict(internal_rest_generate_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


