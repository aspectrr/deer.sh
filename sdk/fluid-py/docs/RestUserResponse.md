# RestUserResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**avatar_url** | **str** |  | [optional] 
**display_name** | **str** |  | [optional] 
**email** | **str** |  | [optional] 
**email_verified** | **bool** |  | [optional] 
**id** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_user_response import RestUserResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestUserResponse from a JSON string
rest_user_response_instance = RestUserResponse.from_json(json)
# print the JSON string representation of the object
print(RestUserResponse.to_json())

# convert the object into a dict
rest_user_response_dict = rest_user_response_instance.to_dict()
# create an instance of RestUserResponse from a dict
rest_user_response_from_dict = RestUserResponse.from_dict(rest_user_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


