# InternalAnsibleJobRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**check** | **bool** |  | [optional] 
**playbook** | **str** |  | [optional] 
**vm_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_job_request import InternalAnsibleJobRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleJobRequest from a JSON string
internal_ansible_job_request_instance = InternalAnsibleJobRequest.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleJobRequest.to_json())

# convert the object into a dict
internal_ansible_job_request_dict = internal_ansible_job_request_instance.to_dict()
# create an instance of InternalAnsibleJobRequest from a dict
internal_ansible_job_request_from_dict = InternalAnsibleJobRequest.from_dict(internal_ansible_job_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


