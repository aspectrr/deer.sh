# InternalAnsibleGetPlaybookResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbook** | [**GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook**](GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook.md) |  | [optional] 
**tasks** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask]**](GithubComAspectrrFluidShFluidRemoteInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_get_playbook_response import InternalAnsibleGetPlaybookResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleGetPlaybookResponse from a JSON string
internal_ansible_get_playbook_response_instance = InternalAnsibleGetPlaybookResponse.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleGetPlaybookResponse.to_json())

# convert the object into a dict
internal_ansible_get_playbook_response_dict = internal_ansible_get_playbook_response_instance.to_dict()
# create an instance of InternalAnsibleGetPlaybookResponse from a dict
internal_ansible_get_playbook_response_from_dict = InternalAnsibleGetPlaybookResponse.from_dict(internal_ansible_get_playbook_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


