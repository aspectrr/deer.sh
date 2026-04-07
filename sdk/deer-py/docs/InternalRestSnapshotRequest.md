# InternalRestSnapshotRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**external** | **bool** | optional; default false (internal snapshot) | [optional] 
**name** | **str** | required | [optional] 

## Example

```python
from fluid.models.internal_rest_snapshot_request import InternalRestSnapshotRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestSnapshotRequest from a JSON string
internal_rest_snapshot_request_instance = InternalRestSnapshotRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestSnapshotRequest.to_json())

# convert the object into a dict
internal_rest_snapshot_request_dict = internal_rest_snapshot_request_instance.to_dict()
# create an instance of InternalRestSnapshotRequest from a dict
internal_rest_snapshot_request_from_dict = InternalRestSnapshotRequest.from_dict(internal_rest_snapshot_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


