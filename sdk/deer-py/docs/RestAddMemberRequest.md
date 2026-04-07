# RestAddMemberRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**email** | **str** |  | [optional] 
**role** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_add_member_request import RestAddMemberRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestAddMemberRequest from a JSON string
rest_add_member_request_instance = RestAddMemberRequest.from_json(json)
# print the JSON string representation of the object
print(RestAddMemberRequest.to_json())

# convert the object into a dict
rest_add_member_request_dict = rest_add_member_request_instance.to_dict()
# create an instance of RestAddMemberRequest from a dict
rest_add_member_request_from_dict = RestAddMemberRequest.from_dict(rest_add_member_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


