# FluidRemoteInternalAnsibleAddTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **object** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_add_task_request import FluidRemoteInternalAnsibleAddTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleAddTaskRequest from a JSON string
fluid_remote_internal_ansible_add_task_request_instance = FluidRemoteInternalAnsibleAddTaskRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleAddTaskRequest.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_add_task_request_dict = fluid_remote_internal_ansible_add_task_request_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleAddTaskRequest from a dict
fluid_remote_internal_ansible_add_task_request_from_dict = FluidRemoteInternalAnsibleAddTaskRequest.from_dict(fluid_remote_internal_ansible_add_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


