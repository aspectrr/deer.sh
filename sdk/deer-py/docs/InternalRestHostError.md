# InternalRestHostError


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | **str** |  | [optional] 
**host_address** | **str** |  | [optional] 
**host_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_host_error import InternalRestHostError

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestHostError from a JSON string
internal_rest_host_error_instance = InternalRestHostError.from_json(json)
# print the JSON string representation of the object
print(InternalRestHostError.to_json())

# convert the object into a dict
internal_rest_host_error_dict = internal_rest_host_error_instance.to_dict()
# create an instance of InternalRestHostError from a dict
internal_rest_host_error_from_dict = InternalRestHostError.from_dict(internal_rest_host_error_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


