# OrchestratorHostInfo


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_sandboxes** | **int** |  | [optional] 
**available_cpus** | **int** |  | [optional] 
**available_disk_mb** | **int** |  | [optional] 
**available_memory_mb** | **int** |  | [optional] 
**base_images** | **List[str]** |  | [optional] 
**host_id** | **str** |  | [optional] 
**hostname** | **str** |  | [optional] 
**last_heartbeat** | **str** |  | [optional] 
**status** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_host_info import OrchestratorHostInfo

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorHostInfo from a JSON string
orchestrator_host_info_instance = OrchestratorHostInfo.from_json(json)
# print the JSON string representation of the object
print(OrchestratorHostInfo.to_json())

# convert the object into a dict
orchestrator_host_info_dict = orchestrator_host_info_instance.to_dict()
# create an instance of OrchestratorHostInfo from a dict
orchestrator_host_info_from_dict = OrchestratorHostInfo.from_dict(orchestrator_host_info_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


