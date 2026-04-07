# GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**host_errors** | [**List[GithubComAspectrrFluidShFluidRemoteInternalRestHostError]**](GithubComAspectrrFluidShFluidRemoteInternalRestHostError.md) | Errors from unreachable hosts (multi-host mode) | [optional] 
**vms** | [**List[GithubComAspectrrFluidShFluidRemoteInternalRestVmInfo]**](GithubComAspectrrFluidShFluidRemoteInternalRestVmInfo.md) |  | [optional] 

## Example

```python
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response import GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse

# TODO update the JSON string below
json = "{}"
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse from a JSON string
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response_instance = GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse.from_json(json)
# print the JSON string representation of the object
print(GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse.to_json())

# convert the object into a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response_dict = github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response_instance.to_dict()
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse from a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response_from_dict = GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse.from_dict(github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


