# InternalRestGetSandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStoreCommand]**](GithubComAspectrrFluidShFluidRemoteInternalStoreCommand.md) |  | [optional] 
**sandbox** | [**GithubComAspectrrFluidShFluidRemoteInternalStoreSandbox**](GithubComAspectrrFluidShFluidRemoteInternalStoreSandbox.md) |  | [optional] 

## Example

```python
from fluid.models.internal_rest_get_sandbox_response import InternalRestGetSandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestGetSandboxResponse from a JSON string
internal_rest_get_sandbox_response_instance = InternalRestGetSandboxResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestGetSandboxResponse.to_json())

# convert the object into a dict
internal_rest_get_sandbox_response_dict = internal_rest_get_sandbox_response_instance.to_dict()
# create an instance of InternalRestGetSandboxResponse from a dict
internal_rest_get_sandbox_response_from_dict = InternalRestGetSandboxResponse.from_dict(internal_rest_get_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


