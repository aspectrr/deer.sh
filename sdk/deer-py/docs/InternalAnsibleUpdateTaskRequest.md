# InternalAnsibleUpdateTaskRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**module** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**params** | **object** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_update_task_request import InternalAnsibleUpdateTaskRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleUpdateTaskRequest from a JSON string
internal_ansible_update_task_request_instance = InternalAnsibleUpdateTaskRequest.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleUpdateTaskRequest.to_json())

# convert the object into a dict
internal_ansible_update_task_request_dict = internal_ansible_update_task_request_instance.to_dict()
# create an instance of InternalAnsibleUpdateTaskRequest from a dict
internal_ansible_update_task_request_from_dict = InternalAnsibleUpdateTaskRequest.from_dict(internal_ansible_update_task_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


