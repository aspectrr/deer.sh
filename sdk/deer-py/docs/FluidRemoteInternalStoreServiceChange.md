# FluidRemoteInternalStoreServiceChange


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  | [optional] 
**name** | **str** |  | [optional] 
**state** | **str** | started|stopped|restarted|reloaded | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_service_change import FluidRemoteInternalStoreServiceChange

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreServiceChange from a JSON string
fluid_remote_internal_store_service_change_instance = FluidRemoteInternalStoreServiceChange.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreServiceChange.to_json())

# convert the object into a dict
fluid_remote_internal_store_service_change_dict = fluid_remote_internal_store_service_change_instance.to_dict()
# create an instance of FluidRemoteInternalStoreServiceChange from a dict
fluid_remote_internal_store_service_change_from_dict = FluidRemoteInternalStoreServiceChange.from_dict(fluid_remote_internal_store_service_change_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


