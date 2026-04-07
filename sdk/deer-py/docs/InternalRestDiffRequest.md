# InternalRestDiffRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**from_snapshot** | **str** | required | [optional] 
**to_snapshot** | **str** | required | [optional] 

## Example

```python
from fluid.models.internal_rest_diff_request import InternalRestDiffRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestDiffRequest from a JSON string
internal_rest_diff_request_instance = InternalRestDiffRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestDiffRequest.to_json())

# convert the object into a dict
internal_rest_diff_request_dict = internal_rest_diff_request_instance.to_dict()
# create an instance of InternalRestDiffRequest from a dict
internal_rest_diff_request_from_dict = InternalRestDiffRequest.from_dict(internal_rest_diff_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


