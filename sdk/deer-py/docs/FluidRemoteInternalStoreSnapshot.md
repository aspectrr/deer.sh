# FluidRemoteInternalStoreSnapshot


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**kind** | [**FluidRemoteInternalStoreSnapshotKind**](FluidRemoteInternalStoreSnapshotKind.md) |  | [optional] 
**meta_json** | **str** | optional JSON metadata | [optional] 
**name** | **str** | logical name (unique per sandbox) | [optional] 
**ref** | **str** | Ref is a backend-specific reference: for internal snapshots this could be a UUID or name, for external snapshots it could be a file path to the overlay qcow2. | [optional] 
**sandbox_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_store_snapshot import FluidRemoteInternalStoreSnapshot

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalStoreSnapshot from a JSON string
fluid_remote_internal_store_snapshot_instance = FluidRemoteInternalStoreSnapshot.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalStoreSnapshot.to_json())

# convert the object into a dict
fluid_remote_internal_store_snapshot_dict = fluid_remote_internal_store_snapshot_instance.to_dict()
# create an instance of FluidRemoteInternalStoreSnapshot from a dict
fluid_remote_internal_store_snapshot_from_dict = FluidRemoteInternalStoreSnapshot.from_dict(fluid_remote_internal_store_snapshot_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


