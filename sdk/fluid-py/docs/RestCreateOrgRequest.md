# RestCreateOrgRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**slug** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_create_org_request import RestCreateOrgRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestCreateOrgRequest from a JSON string
rest_create_org_request_instance = RestCreateOrgRequest.from_json(json)
# print the JSON string representation of the object
print(RestCreateOrgRequest.to_json())

# convert the object into a dict
rest_create_org_request_dict = rest_create_org_request_instance.to_dict()
# create an instance of RestCreateOrgRequest from a dict
rest_create_org_request_from_dict = RestCreateOrgRequest.from_dict(rest_create_org_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


