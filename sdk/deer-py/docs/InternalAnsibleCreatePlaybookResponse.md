# InternalAnsibleCreatePlaybookResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbook** | [**GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook**](GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook.md) |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_create_playbook_response import InternalAnsibleCreatePlaybookResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleCreatePlaybookResponse from a JSON string
internal_ansible_create_playbook_response_instance = InternalAnsibleCreatePlaybookResponse.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleCreatePlaybookResponse.to_json())

# convert the object into a dict
internal_ansible_create_playbook_response_dict = internal_ansible_create_playbook_response_instance.to_dict()
# create an instance of InternalAnsibleCreatePlaybookResponse from a dict
internal_ansible_create_playbook_response_from_dict = InternalAnsibleCreatePlaybookResponse.from_dict(internal_ansible_create_playbook_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


