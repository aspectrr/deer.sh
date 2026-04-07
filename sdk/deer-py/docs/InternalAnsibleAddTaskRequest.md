# InternalAnsibleAddTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **object** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_add_task_request import InternalAnsibleAddTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleAddTaskRequest from a JSON string
internal_ansible_add_task_request_instance = InternalAnsibleAddTaskRequest.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleAddTaskRequest.to_json())

# convert the object into a dict
internal_ansible_add_task_request_dict = internal_ansible_add_task_request_instance.to_dict()
# create an instance of InternalAnsibleAddTaskRequest from a dict
internal_ansible_add_task_request_from_dict = InternalAnsibleAddTaskRequest.from_dict(internal_ansible_add_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


