# GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ip_address** | **str** | populated when auto_start and wait_for_ip are true | [optional] 
**sandbox** | [**GithubComAspectrrFluidShFluidRemoteInternalStoreSandbox**](GithubComAspectrrFluidShFluidRemoteInternalStoreSandbox.md) |  | [optional] 

## Example

```python
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response import GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse

# TODO update the JSON string below
json = "{}"
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse from a JSON string
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response_instance = GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse.from_json(json)
# print the JSON string representation of the object
print(GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse.to_json())

# convert the object into a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response_dict = github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response_instance.to_dict()
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse from a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response_from_dict = GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse.from_dict(github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


