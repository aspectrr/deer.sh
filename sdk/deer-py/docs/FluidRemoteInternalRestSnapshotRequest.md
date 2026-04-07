# FluidRemoteInternalRestSnapshotRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**external** | **bool** | optional; default false (internal snapshot) | [optional] 
**name** | **str** | required | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_snapshot_request import FluidRemoteInternalRestSnapshotRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestSnapshotRequest from a JSON string
fluid_remote_internal_rest_snapshot_request_instance = FluidRemoteInternalRestSnapshotRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestSnapshotRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_snapshot_request_dict = fluid_remote_internal_rest_snapshot_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestSnapshotRequest from a dict
fluid_remote_internal_rest_snapshot_request_from_dict = FluidRemoteInternalRestSnapshotRequest.from_dict(fluid_remote_internal_rest_snapshot_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


