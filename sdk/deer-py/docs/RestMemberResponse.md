# RestMemberResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**role** | **str** |  | [optional] 
**user_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_member_response import RestMemberResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestMemberResponse from a JSON string
rest_member_response_instance = RestMemberResponse.from_json(json)
# print the JSON string representation of the object
print(RestMemberResponse.to_json())

# convert the object into a dict
rest_member_response_dict = rest_member_response_instance.to_dict()
# create an instance of RestMemberResponse from a dict
rest_member_response_from_dict = RestMemberResponse.from_dict(rest_member_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


