# InternalAnsibleReorderTasksRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**task_ids** | **List[str]** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_reorder_tasks_request import InternalAnsibleReorderTasksRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleReorderTasksRequest from a JSON string
internal_ansible_reorder_tasks_request_instance = InternalAnsibleReorderTasksRequest.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleReorderTasksRequest.to_json())

# convert the object into a dict
internal_ansible_reorder_tasks_request_dict = internal_ansible_reorder_tasks_request_instance.to_dict()
# create an instance of InternalAnsibleReorderTasksRequest from a dict
internal_ansible_reorder_tasks_request_from_dict = InternalAnsibleReorderTasksRequest.from_dict(internal_ansible_reorder_tasks_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


