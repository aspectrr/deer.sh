# InternalRestSandboxInfo


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_id** | **str** |  | [optional] 
**base_image** | **str** |  | [optional] 
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**ip_address** | **str** |  | [optional] 
**job_id** | **str** |  | [optional] 
**network** | **str** |  | [optional] 
**sandbox_name** | **str** |  | [optional] 
**state** | **str** |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 
**updated_at** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_sandbox_info import InternalRestSandboxInfo

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSandboxInfo from a JSON string
internal_rest_sandbox_info_instance = InternalRestSandboxInfo.from_json(json)
# print the JSON string representation of the object
print(InternalRestSandboxInfo.to_json())

# convert the object into a dict
internal_rest_sandbox_info_dict = internal_rest_sandbox_info_instance.to_dict()
# create an instance of InternalRestSandboxInfo from a dict
internal_rest_sandbox_info_from_dict = InternalRestSandboxInfo.from_dict(internal_rest_sandbox_info_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


