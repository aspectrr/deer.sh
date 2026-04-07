# FluidRemoteInternalRestPublishRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_id** | **str** | required | [optional] 
**message** | **str** | optional commit/PR message | [optional] 
**reviewers** | **List[str]** | optional | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_publish_request import FluidRemoteInternalRestPublishRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestPublishRequest from a JSON string
fluid_remote_internal_rest_publish_request_instance = FluidRemoteInternalRestPublishRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestPublishRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_publish_request_dict = fluid_remote_internal_rest_publish_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestPublishRequest from a dict
fluid_remote_internal_rest_publish_request_from_dict = FluidRemoteInternalRestPublishRequest.from_dict(fluid_remote_internal_rest_publish_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


