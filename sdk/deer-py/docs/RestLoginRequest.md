# RestLoginRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**email** | **str** |  | [optional] 
**password** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_login_request import RestLoginRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestLoginRequest from a JSON string
rest_login_request_instance = RestLoginRequest.from_json(json)
# print the JSON string representation of the object
print(RestLoginRequest.to_json())

# convert the object into a dict
rest_login_request_dict = rest_login_request_instance.to_dict()
# create an instance of RestLoginRequest from a dict
rest_login_request_from_dict = RestLoginRequest.from_dict(rest_login_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


