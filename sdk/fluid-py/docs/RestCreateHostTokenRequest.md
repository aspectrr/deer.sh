# RestCreateHostTokenRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_create_host_token_request import RestCreateHostTokenRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestCreateHostTokenRequest from a JSON string
rest_create_host_token_request_instance = RestCreateHostTokenRequest.from_json(json)
# print the JSON string representation of the object
print(RestCreateHostTokenRequest.to_json())

# convert the object into a dict
rest_create_host_token_request_dict = rest_create_host_token_request_instance.to_dict()
# create an instance of RestCreateHostTokenRequest from a dict
rest_create_host_token_request_from_dict = RestCreateHostTokenRequest.from_dict(rest_create_host_token_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


