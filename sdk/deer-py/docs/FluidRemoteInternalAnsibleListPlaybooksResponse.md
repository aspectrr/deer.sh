# FluidRemoteInternalAnsibleListPlaybooksResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbooks** | [**List[FluidRemoteInternalStorePlaybook]**](FluidRemoteInternalStorePlaybook.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_list_playbooks_response import FluidRemoteInternalAnsibleListPlaybooksResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleListPlaybooksResponse from a JSON string
fluid_remote_internal_ansible_list_playbooks_response_instance = FluidRemoteInternalAnsibleListPlaybooksResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleListPlaybooksResponse.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_list_playbooks_response_dict = fluid_remote_internal_ansible_list_playbooks_response_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleListPlaybooksResponse from a dict
fluid_remote_internal_ansible_list_playbooks_response_from_dict = FluidRemoteInternalAnsibleListPlaybooksResponse.from_dict(fluid_remote_internal_ansible_list_playbooks_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


