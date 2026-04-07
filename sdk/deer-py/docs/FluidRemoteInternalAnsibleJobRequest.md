# FluidRemoteInternalAnsibleJobRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**check** | **bool** |  | [optional] 
**playbook** | **str** |  | [optional] 
**vm_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_job_request import FluidRemoteInternalAnsibleJobRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleJobRequest from a JSON string
fluid_remote_internal_ansible_job_request_instance = FluidRemoteInternalAnsibleJobRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleJobRequest.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_job_request_dict = fluid_remote_internal_ansible_job_request_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleJobRequest from a dict
fluid_remote_internal_ansible_job_request_from_dict = FluidRemoteInternalAnsibleJobRequest.from_dict(fluid_remote_internal_ansible_job_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


