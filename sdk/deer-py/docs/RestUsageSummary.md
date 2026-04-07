# RestUsageSummary


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_hosts** | **float** |  | [optional] 
**sandbox_hours** | **float** |  | [optional] 
**source_vms** | **float** |  | [optional] 

## Example

```python
from fluid.models.rest_usage_summary import RestUsageSummary

# TODO update the JSON string below
json = "{}"
# create an instance of RestUsageSummary from a JSON string
rest_usage_summary_instance = RestUsageSummary.from_json(json)
# print the JSON string representation of the object
print(RestUsageSummary.to_json())

# convert the object into a dict
rest_usage_summary_dict = rest_usage_summary_instance.to_dict()
# create an instance of RestUsageSummary from a dict
rest_usage_summary_from_dict = RestUsageSummary.from_dict(rest_usage_summary_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


