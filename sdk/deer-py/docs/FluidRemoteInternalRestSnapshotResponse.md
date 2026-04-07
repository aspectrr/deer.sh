# FluidRemoteInternalRestSnapshotResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**snapshot** | [**FluidRemoteInternalStoreSnapshot**](FluidRemoteInternalStoreSnapshot.md) |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_snapshot_response import FluidRemoteInternalRestSnapshotResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSnapshotResponse from a JSON string
fluid_remote_internal_rest_snapshot_response_instance = FluidRemoteInternalRestSnapshotResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSnapshotResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_snapshot_response_dict = fluid_remote_internal_rest_snapshot_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestSnapshotResponse from a dict
fluid_remote_internal_rest_snapshot_response_from_dict = FluidRemoteInternalRestSnapshotResponse.from_dict(fluid_remote_internal_rest_snapshot_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


