# InternalRestDiffResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**diff** | [**GithubComAspectrrFluidShFluidRemoteInternalStoreDiff**](GithubComAspectrrFluidShFluidRemoteInternalStoreDiff.md) |  | [optional] 

## Example

```python
from fluid.models.internal_rest_diff_response import InternalRestDiffResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestDiffResponse from a JSON string
internal_rest_diff_response_instance = InternalRestDiffResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestDiffResponse.to_json())

# convert the object into a dict
internal_rest_diff_response_dict = internal_rest_diff_response_instance.to_dict()
# create an instance of InternalRestDiffResponse from a dict
internal_rest_diff_response_from_dict = InternalRestDiffResponse.from_dict(internal_rest_diff_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


