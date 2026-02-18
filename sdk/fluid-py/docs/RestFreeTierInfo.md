# RestFreeTierInfo


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**max_agent_hosts** | **int** |  | [optional] 
**max_concurrent_sandboxes** | **int** |  | [optional] 
**max_source_vms** | **int** |  | [optional] 

## Example

```python
from fluid.models.rest_free_tier_info import RestFreeTierInfo

# TODO update the JSON string below
json = "{}"
# create an instance of RestFreeTierInfo from a JSON string
rest_free_tier_info_instance = RestFreeTierInfo.from_json(json)
# print the JSON string representation of the object
print(RestFreeTierInfo.to_json())

# convert the object into a dict
rest_free_tier_info_dict = rest_free_tier_info_instance.to_dict()
# create an instance of RestFreeTierInfo from a dict
rest_free_tier_info_from_dict = RestFreeTierInfo.from_dict(rest_free_tier_info_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


