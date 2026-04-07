# InternalAnsibleAddTaskResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task** | [**GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask**](GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_add_task_response import InternalAnsibleAddTaskResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleAddTaskResponse from a JSON string
internal_ansible_add_task_response_instance = InternalAnsibleAddTaskResponse.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleAddTaskResponse.to_json())

# convert the object into a dict
internal_ansible_add_task_response_dict = internal_ansible_add_task_response_instance.to_dict()
# create an instance of InternalAnsibleAddTaskResponse from a dict
internal_ansible_add_task_response_from_dict = InternalAnsibleAddTaskResponse.from_dict(internal_ansible_add_task_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


