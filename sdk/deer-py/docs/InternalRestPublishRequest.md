# InternalRestPublishRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**job_id** | **str** | required | [optional] 
**message** | **str** | optional commit/PR message | [optional] 
**reviewers** | **List[str]** | optional | [optional] 

## Example

```python
from fluid.models.internal_rest_publish_request import InternalRestPublishRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestPublishRequest from a JSON string
internal_rest_publish_request_instance = InternalRestPublishRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestPublishRequest.to_json())

# convert the object into a dict
internal_rest_publish_request_dict = internal_rest_publish_request_instance.to_dict()
# create an instance of InternalRestPublishRequest from a dict
internal_rest_publish_request_from_dict = InternalRestPublishRequest.from_dict(internal_rest_publish_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


