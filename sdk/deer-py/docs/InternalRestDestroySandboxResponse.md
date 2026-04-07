# InternalRestDestroySandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**base_image** | **str** |  | [optional] 
**sandbox_name** | **str** |  | [optional] 
**state** | [**GithubComAspectrrFluidShFluidRemoteInternalStoreSandboxState**](GithubComAspectrrFluidShFluidRemoteInternalStoreSandboxState.md) |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_destroy_sandbox_response import InternalRestDestroySandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestDestroySandboxResponse from a JSON string
internal_rest_destroy_sandbox_response_instance = InternalRestDestroySandboxResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestDestroySandboxResponse.to_json())

# convert the object into a dict
internal_rest_destroy_sandbox_response_dict = internal_rest_destroy_sandbox_response_instance.to_dict()
# create an instance of InternalRestDestroySandboxResponse from a dict
internal_rest_destroy_sandbox_response_from_dict = InternalRestDestroySandboxResponse.from_dict(internal_rest_destroy_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


