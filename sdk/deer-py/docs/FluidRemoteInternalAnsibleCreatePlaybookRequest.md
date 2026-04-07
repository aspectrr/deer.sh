# FluidRemoteInternalAnsibleCreatePlaybookRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**become** | **bool** |  | [optional] 
**hosts** | **str** |  | [optional] 
**name** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_create_playbook_request import FluidRemoteInternalAnsibleCreatePlaybookRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleCreatePlaybookRequest from a JSON string
fluid_remote_internal_ansible_create_playbook_request_instance = FluidRemoteInternalAnsibleCreatePlaybookRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleCreatePlaybookRequest.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_create_playbook_request_dict = fluid_remote_internal_ansible_create_playbook_request_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleCreatePlaybookRequest from a dict
fluid_remote_internal_ansible_create_playbook_request_from_dict = FluidRemoteInternalAnsibleCreatePlaybookRequest.from_dict(fluid_remote_internal_ansible_create_playbook_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


