# InternalAnsibleJob


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**check** | **bool** |  | [optional] 
**id** | **str** |  | [optional] 
**playbook** | **str** |  | [optional] 
**status** | [**InternalAnsibleJobStatus**](InternalAnsibleJobStatus.md) |  | [optional] 
**vm_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_job import InternalAnsibleJob

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleJob from a JSON string
internal_ansible_job_instance = InternalAnsibleJob.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleJob.to_json())

# convert the object into a dict
internal_ansible_job_dict = internal_ansible_job_instance.to_dict()
# create an instance of InternalAnsibleJob from a dict
internal_ansible_job_from_dict = InternalAnsibleJob.from_dict(internal_ansible_job_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


