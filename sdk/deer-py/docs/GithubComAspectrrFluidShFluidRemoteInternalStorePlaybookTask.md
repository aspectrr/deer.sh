# GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**module** | **str** | ansible module (apt, shell, copy, etc.) | [optional] 
**name** | **str** | task name/description | [optional] 
**params** | **object** | module-specific parameters | [optional] 
**playbook_id** | **str** |  | [optional] 
**position** | **int** | ordering within playbook | [optional] 

## Example

```python
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task import GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask

# TODO update the JSON string below
json = "{}"
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask from a JSON string
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task_instance = GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask.from_json(json)
# print the JSON string representation of the object
print(GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask.to_json())

# convert the object into a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task_dict = github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task_instance.to_dict()
# create an instance of GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask from a dict
github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task_from_dict = GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask.from_dict(github_com_aspectrr_fluid_sh_fluid_remote_internal_store_playbook_task_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


