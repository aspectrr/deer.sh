# OrchestratorSourceCommandResult


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**exit_code** | **int** |  | [optional] 
**source_vm** | **str** |  | [optional] 
**stderr** | **str** |  | [optional] 
**stdout** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_source_command_result import OrchestratorSourceCommandResult

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorSourceCommandResult from a JSON string
orchestrator_source_command_result_instance = OrchestratorSourceCommandResult.from_json(json)
# print the JSON string representation of the object
print(OrchestratorSourceCommandResult.to_json())

# convert the object into a dict
orchestrator_source_command_result_dict = orchestrator_source_command_result_instance.to_dict()
# create an instance of OrchestratorSourceCommandResult from a dict
orchestrator_source_command_result_from_dict = OrchestratorSourceCommandResult.from_dict(orchestrator_source_command_result_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


