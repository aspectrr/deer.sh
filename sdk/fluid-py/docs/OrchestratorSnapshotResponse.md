# OrchestratorSnapshotResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**created_at** | **str** |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**snapshot_id** | **str** |  | [optional] 
**snapshot_name** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_snapshot_response import OrchestratorSnapshotResponse

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorSnapshotResponse from a JSON string
orchestrator_snapshot_response_instance = OrchestratorSnapshotResponse.from_json(json)
# print the JSON string representation of the object
print(OrchestratorSnapshotResponse.to_json())

# convert the object into a dict
orchestrator_snapshot_response_dict = orchestrator_snapshot_response_instance.to_dict()
# create an instance of OrchestratorSnapshotResponse from a dict
orchestrator_snapshot_response_from_dict = OrchestratorSnapshotResponse.from_dict(orchestrator_snapshot_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


