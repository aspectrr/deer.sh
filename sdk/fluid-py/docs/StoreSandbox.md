# StoreSandbox


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_id** | **str** |  | [optional] 
**base_image** | **str** |  | [optional] 
**bridge** | **str** |  | [optional] 
**created_at** | **str** |  | [optional] 
**deleted_at** | **str** |  | [optional] 
**host_id** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**ip_address** | **str** |  | [optional] 
**mac_address** | **str** |  | [optional] 
**memory_mb** | **int** |  | [optional] 
**name** | **str** |  | [optional] 
**org_id** | **str** |  | [optional] 
**source_vm** | **str** |  | [optional] 
**state** | [**StoreSandboxState**](StoreSandboxState.md) |  | [optional] 
**tap_device** | **str** |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 
**updated_at** | **str** |  | [optional] 
**vcpus** | **int** |  | [optional] 

## Example

```python
from fluid.models.store_sandbox import StoreSandbox

# TODO update the JSON string below
json = "{}"
# create an instance of StoreSandbox from a JSON string
store_sandbox_instance = StoreSandbox.from_json(json)
# print the JSON string representation of the object
print(StoreSandbox.to_json())

# convert the object into a dict
store_sandbox_dict = store_sandbox_instance.to_dict()
# create an instance of StoreSandbox from a dict
store_sandbox_from_dict = StoreSandbox.from_dict(store_sandbox_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


