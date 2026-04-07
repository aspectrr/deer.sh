# InternalRestPublishResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**note** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_publish_response import InternalRestPublishResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestPublishResponse from a JSON string
internal_rest_publish_response_instance = InternalRestPublishResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestPublishResponse.to_json())

# convert the object into a dict
internal_rest_publish_response_dict = internal_rest_publish_response_instance.to_dict()
# create an instance of InternalRestPublishResponse from a dict
internal_rest_publish_response_from_dict = InternalRestPublishResponse.from_dict(internal_rest_publish_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


