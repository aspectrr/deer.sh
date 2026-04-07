# RestUpdateOrgRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_update_org_request import RestUpdateOrgRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestUpdateOrgRequest from a JSON string
rest_update_org_request_instance = RestUpdateOrgRequest.from_json(json)
# print the JSON string representation of the object
print(RestUpdateOrgRequest.to_json())

# convert the object into a dict
rest_update_org_request_dict = rest_update_org_request_instance.to_dict()
# create an instance of RestUpdateOrgRequest from a dict
rest_update_org_request_from_dict = RestUpdateOrgRequest.from_dict(rest_update_org_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


