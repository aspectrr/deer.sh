# GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**commands_run** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStoreCommandSummary]**](GithubComAspectrrFluidShFluidRemoteInternalStoreCommandSummary.md) |  | [optional] 
**files_added** | **List[str]** |  | [optional] 
**files_modified** | **List[str]** |  | [optional] 
**files_removed** | **List[str]** |  | [optional] 
**packages_added** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStorePackageInfo]**](GithubComAspectrrFluidShFluidRemoteInternalStorePackageInfo.md) |  | [optional] 
**packages_removed** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStorePackageInfo]**](GithubComAspectrrFluidShFluidRemoteInternalStorePackageInfo.md) |  | [optional] 
**services_changed** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStoreServiceChange]**](GithubComAspectrrFluidShFluidRemoteInternalStoreServiceChange.md) |  | [optional] 

## Example

```python
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff import GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff

# TODO update the JSON string below
json = "{}"
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff from a JSON string
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff_instance = GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff.from_json(json)
# print the JSON string representation of the object
print(GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff.to_json())

# convert the object into a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff_dict = github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff_instance.to_dict()
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff from a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff_from_dict = GithubComAspectrrFluidShFluidRemoteInternalStoreChangeDiff.from_dict(github_com_aspectrr_fluid_sh_fluid_remote_internal_store_change_diff_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


