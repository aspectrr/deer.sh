# RestRegisterRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**display_name** | **str** |  | [optional] 
**email** | **str** |  | [optional] 
**password** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_register_request import RestRegisterRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestRegisterRequest from a JSON string
rest_register_request_instance = RestRegisterRequest.from_json(json)
# print the JSON string representation of the object
print(RestRegisterRequest.to_json())

# convert the object into a dict
rest_register_request_dict = rest_register_request_instance.to_dict()
# create an instance of RestRegisterRequest from a dict
rest_register_request_from_dict = RestRegisterRequest.from_dict(rest_register_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


