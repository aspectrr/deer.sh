# RestBillingResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**free_tier** | [**RestFreeTierInfo**](RestFreeTierInfo.md) |  | [optional] 
**plan** | **str** |  | [optional] 
**status** | **str** |  | [optional] 
**usage** | [**RestUsageSummary**](RestUsageSummary.md) |  | [optional] 

## Example

```python
from fluid.models.rest_billing_response import RestBillingResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestBillingResponse from a JSON string
rest_billing_response_instance = RestBillingResponse.from_json(json)
# print the JSON string representation of the object
print(RestBillingResponse.to_json())

# convert the object into a dict
rest_billing_response_dict = rest_billing_response_instance.to_dict()
# create an instance of RestBillingResponse from a dict
rest_billing_response_from_dict = RestBillingResponse.from_dict(rest_billing_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


