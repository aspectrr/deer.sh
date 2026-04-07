# GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** | required | [optional] 
**env** | **Dict[str, str]** | optional | [optional] 
**private_key_path** | **str** | optional; if empty, uses managed credentials (requires SSH CA) | [optional] 
**timeout_sec** | **int** | optional; default from service config | [optional] 
**user** | **str** | optional; defaults to \&quot;sandbox\&quot; when using managed credentials | [optional] 

## Example

```python
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request import GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest

# TODO update the JSON string below
json = "{}"
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest from a JSON string
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request_instance = GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest.from_json(json)
# print the JSON string representation of the object
print(GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest.to_json())

# convert the object into a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request_dict = github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request_instance.to_dict()
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest from a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request_from_dict = GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest.from_dict(github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


