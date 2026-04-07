# InternalAnsibleCreatePlaybookRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**become** | **bool** |  | [optional] 
**hosts** | **str** |  | [optional] 
**name** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_create_playbook_request import InternalAnsibleCreatePlaybookRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleCreatePlaybookRequest from a JSON string
internal_ansible_create_playbook_request_instance = InternalAnsibleCreatePlaybookRequest.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleCreatePlaybookRequest.to_json())

# convert the object into a dict
internal_ansible_create_playbook_request_dict = internal_ansible_create_playbook_request_instance.to_dict()
# create an instance of InternalAnsibleCreatePlaybookRequest from a dict
internal_ansible_create_playbook_request_from_dict = InternalAnsibleCreatePlaybookRequest.from_dict(internal_ansible_create_playbook_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


