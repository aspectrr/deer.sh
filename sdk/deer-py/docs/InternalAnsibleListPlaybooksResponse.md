# InternalAnsibleListPlaybooksResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbooks** | [**List[GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook]**](GithubComAspectrrFluidShFluidRemoteInternalStorePlaybook.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_list_playbooks_response import InternalAnsibleListPlaybooksResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleListPlaybooksResponse from a JSON string
internal_ansible_list_playbooks_response_instance = InternalAnsibleListPlaybooksResponse.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleListPlaybooksResponse.to_json())

# convert the object into a dict
internal_ansible_list_playbooks_response_dict = internal_ansible_list_playbooks_response_instance.to_dict()
# create an instance of InternalAnsibleListPlaybooksResponse from a dict
internal_ansible_list_playbooks_response_from_dict = InternalAnsibleListPlaybooksResponse.from_dict(internal_ansible_list_playbooks_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


