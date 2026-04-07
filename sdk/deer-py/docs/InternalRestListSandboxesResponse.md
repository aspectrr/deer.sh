# InternalRestListSandboxesResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**sandboxes** | [**List[InternalRestSandboxInfo]**](InternalRestSandboxInfo.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_list_sandboxes_response import InternalRestListSandboxesResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestListSandboxesResponse from a JSON string
internal_rest_list_sandboxes_response_instance = InternalRestListSandboxesResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestListSandboxesResponse.to_json())

# convert the object into a dict
internal_rest_list_sandboxes_response_dict = internal_rest_list_sandboxes_response_instance.to_dict()
# create an instance of InternalRestListSandboxesResponse from a dict
internal_rest_list_sandboxes_response_from_dict = InternalRestListSandboxesResponse.from_dict(internal_rest_list_sandboxes_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


