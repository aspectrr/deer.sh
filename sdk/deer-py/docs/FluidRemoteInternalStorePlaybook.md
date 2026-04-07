# FluidRemoteInternalStorePlaybook


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**become** | **bool** | whether to use privilege escalation | [optional] 
**created_at** | **str** |  | [optional] 
**file_path** | **str** | rendered YAML file path | [optional] 
**hosts** | **str** | target hosts pattern (e.g., \&quot;all\&quot;, \&quot;webservers\&quot;) | [optional] 
**id** | **str** |  | [optional] 
**name** | **str** | unique playbook name | [optional] 
**updated_at** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_playbook import FluidRemoteInternalStorePlaybook

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStorePlaybook from a JSON string
fluid_remote_internal_store_playbook_instance = FluidRemoteInternalStorePlaybook.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStorePlaybook.to_json())

# convert the object into a dict
fluid_remote_internal_store_playbook_dict = fluid_remote_internal_store_playbook_instance.to_dict()
# create an instance of FluidRemoteInternalStorePlaybook from a dict
fluid_remote_internal_store_playbook_from_dict = FluidRemoteInternalStorePlaybook.from_dict(fluid_remote_internal_store_playbook_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


