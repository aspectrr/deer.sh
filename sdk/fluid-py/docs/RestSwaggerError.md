# RestSwaggerError


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **int** |  | [optional] 
**details** | **str** |  | [optional] 
**error** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_swagger_error import RestSwaggerError

# TODO update the JSON string below
json = "{}"
# create an instance of RestSwaggerError from a JSON string
rest_swagger_error_instance = RestSwaggerError.from_json(json)
# print the JSON string representation of the object
print(RestSwaggerError.to_json())

# convert the object into a dict
rest_swagger_error_dict = rest_swagger_error_instance.to_dict()
# create an instance of RestSwaggerError from a dict
rest_swagger_error_from_dict = RestSwaggerError.from_dict(rest_swagger_error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


