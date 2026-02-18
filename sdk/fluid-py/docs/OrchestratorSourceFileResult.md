# OrchestratorSourceFileResult


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**content** | **str** |  | [optional] 
**path** | **str** |  | [optional] 
**source_vm** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_source_file_result import OrchestratorSourceFileResult

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorSourceFileResult from a JSON string
orchestrator_source_file_result_instance = OrchestratorSourceFileResult.from_json(json)
# print the JSON string representation of the object
print(OrchestratorSourceFileResult.to_json())

# convert the object into a dict
orchestrator_source_file_result_dict = orchestrator_source_file_result_instance.to_dict()
# create an instance of OrchestratorSourceFileResult from a dict
orchestrator_source_file_result_from_dict = OrchestratorSourceFileResult.from_dict(orchestrator_source_file_result_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


