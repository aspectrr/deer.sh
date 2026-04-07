# InternalRestRunCommandResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | [**GithubComAspectrrFluidShFluidRemoteInternalStoreCommand**](GithubComAspectrrFluidShFluidRemoteInternalStoreCommand.md) |  | [optional] 

## Example

```python
from fluid.models.internal_rest_run_command_response import InternalRestRunCommandResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestRunCommandResponse from a JSON string
internal_rest_run_command_response_instance = InternalRestRunCommandResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestRunCommandResponse.to_json())

# convert the object into a dict
internal_rest_run_command_response_dict = internal_rest_run_command_response_instance.to_dict()
# create an instance of InternalRestRunCommandResponse from a dict
internal_rest_run_command_response_from_dict = InternalRestRunCommandResponse.from_dict(internal_rest_run_command_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


