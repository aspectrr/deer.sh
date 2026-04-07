# InternalAnsibleJobResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_id** | **str** |  | [optional] 
**ws_url** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_ansible_job_response import InternalAnsibleJobResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalAnsibleJobResponse from a JSON string
internal_ansible_job_response_instance = InternalAnsibleJobResponse.from_json(json)
# print the JSON string representation of the object
print(InternalAnsibleJobResponse.to_json())

# convert the object into a dict
internal_ansible_job_response_dict = internal_ansible_job_response_instance.to_dict()
# create an instance of InternalAnsibleJobResponse from a dict
internal_ansible_job_response_from_dict = InternalAnsibleJobResponse.from_dict(internal_ansible_job_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


