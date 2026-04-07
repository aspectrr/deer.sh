# RestOrgResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**owner_id** | **str** |  | [optional] 
**slug** | **str** |  | [optional] 
**stripe_customer_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.rest_org_response import RestOrgResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestOrgResponse from a JSON string
rest_org_response_instance = RestOrgResponse.from_json(json)
# print the JSON string representation of the object
print(RestOrgResponse.to_json())

# convert the object into a dict
rest_org_response_dict = rest_org_response_instance.to_dict()
# create an instance of RestOrgResponse from a dict
rest_org_response_from_dict = RestOrgResponse.from_dict(rest_org_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


