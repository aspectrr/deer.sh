# FluidRemoteInternalAnsibleUpdateTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **object** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_update_task_request import FluidRemoteInternalAnsibleUpdateTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleUpdateTaskRequest from a JSON string
fluid_remote_internal_ansible_update_task_request_instance = FluidRemoteInternalAnsibleUpdateTaskRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleUpdateTaskRequest.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_update_task_request_dict = fluid_remote_internal_ansible_update_task_request_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleUpdateTaskRequest from a dict
fluid_remote_internal_ansible_update_task_request_from_dict = FluidRemoteInternalAnsibleUpdateTaskRequest.from_dict(fluid_remote_internal_ansible_update_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


