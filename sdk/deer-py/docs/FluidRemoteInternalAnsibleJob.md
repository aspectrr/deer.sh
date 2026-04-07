# FluidRemoteInternalAnsibleJob


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**check** | **bool** |  | [optional] 
**id** | **str** |  | [optional] 
**playbook** | **str** |  | [optional] 
**status** | [**FluidRemoteInternalAnsibleJobStatus**](FluidRemoteInternalAnsibleJobStatus.md) |  | [optional] 
**vm_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_job import FluidRemoteInternalAnsibleJob

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleJob from a JSON string
fluid_remote_internal_ansible_job_instance = FluidRemoteInternalAnsibleJob.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleJob.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_job_dict = fluid_remote_internal_ansible_job_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleJob from a dict
fluid_remote_internal_ansible_job_from_dict = FluidRemoteInternalAnsibleJob.from_dict(fluid_remote_internal_ansible_job_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


