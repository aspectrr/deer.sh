# RestHostTokenResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**token** | **str** | Only set on creation. | [optional] 

## Example

```python
from fluid.models.rest_host_token_response import RestHostTokenResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestHostTokenResponse from a JSON string
rest_host_token_response_instance = RestHostTokenResponse.from_json(json)
# print the JSON string representation of the object
print(RestHostTokenResponse.to_json())

# convert the object into a dict
rest_host_token_response_dict = rest_host_token_response_instance.to_dict()
# create an instance of RestHostTokenResponse from a dict
rest_host_token_response_from_dict = RestHostTokenResponse.from_dict(rest_host_token_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


