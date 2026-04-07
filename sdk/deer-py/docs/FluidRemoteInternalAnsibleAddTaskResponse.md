# FluidRemoteInternalAnsibleAddTaskResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task** | [**FluidRemoteInternalStorePlaybookTask**](FluidRemoteInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_add_task_response import FluidRemoteInternalAnsibleAddTaskResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleAddTaskResponse from a JSON string
fluid_remote_internal_ansible_add_task_response_instance = FluidRemoteInternalAnsibleAddTaskResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleAddTaskResponse.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_add_task_response_dict = fluid_remote_internal_ansible_add_task_response_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleAddTaskResponse from a dict
fluid_remote_internal_ansible_add_task_response_from_dict = FluidRemoteInternalAnsibleAddTaskResponse.from_dict(fluid_remote_internal_ansible_add_task_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


