# RestAuthResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**user** | [**RestUserResponse**](RestUserResponse.md) |  | [optional] 

## Example

```python
from fluid.models.rest_auth_response import RestAuthResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestAuthResponse from a JSON string
rest_auth_response_instance = RestAuthResponse.from_json(json)
# print the JSON string representation of the object
print(RestAuthResponse.to_json())

# convert the object into a dict
rest_auth_response_dict = rest_auth_response_instance.to_dict()
# create an instance of RestAuthResponse from a dict
rest_auth_response_from_dict = RestAuthResponse.from_dict(rest_auth_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


