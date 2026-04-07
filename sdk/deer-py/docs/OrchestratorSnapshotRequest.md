# OrchestratorSnapshotRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_snapshot_request import OrchestratorSnapshotRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorSnapshotRequest from a JSON string
orchestrator_snapshot_request_instance = OrchestratorSnapshotRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorSnapshotRequest.to_json())

# convert the object into a dict
orchestrator_snapshot_request_dict = orchestrator_snapshot_request_instance.to_dict()
# create an instance of OrchestratorSnapshotRequest from a dict
orchestrator_snapshot_request_from_dict = OrchestratorSnapshotRequest.from_dict(orchestrator_snapshot_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


