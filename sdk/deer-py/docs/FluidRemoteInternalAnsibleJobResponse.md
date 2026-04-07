# FluidRemoteInternalAnsibleJobResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_id** | **str** |  | [optional] 
**ws_url** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_ansible_job_response import FluidRemoteInternalAnsibleJobResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalAnsibleJobResponse from a JSON string
fluid_remote_internal_ansible_job_response_instance = FluidRemoteInternalAnsibleJobResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalAnsibleJobResponse.to_json())

# convert the object into a dict
fluid_remote_internal_ansible_job_response_dict = fluid_remote_internal_ansible_job_response_instance.to_dict()
# create an instance of FluidRemoteInternalAnsibleJobResponse from a dict
fluid_remote_internal_ansible_job_response_from_dict = FluidRemoteInternalAnsibleJobResponse.from_dict(fluid_remote_internal_ansible_job_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


