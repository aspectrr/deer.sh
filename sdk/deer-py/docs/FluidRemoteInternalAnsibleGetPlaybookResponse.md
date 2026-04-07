# FluidRemoteInternalAnsibleGetPlaybookResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**playbook** | [**FluidRemoteInternalStorePlaybook**](FluidRemoteInternalStorePlaybook.md) |  | [optional] 
**tasks** | [**List[FluidRemoteInternalStorePlaybookTask]**](FluidRemoteInternalStorePlaybookTask.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_get_playbook_response import FluidRemoteInternalAnsibleGetPlaybookResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleGetPlaybookResponse from a JSON string
fluid_remote_internal_ansible_get_playbook_response_instance = FluidRemoteInternalAnsibleGetPlaybookResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleGetPlaybookResponse.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_get_playbook_response_dict = fluid_remote_internal_ansible_get_playbook_response_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleGetPlaybookResponse from a dict
fluid_remote_internal_ansible_get_playbook_response_from_dict = FluidRemoteInternalAnsibleGetPlaybookResponse.from_dict(fluid_remote_internal_ansible_get_playbook_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


