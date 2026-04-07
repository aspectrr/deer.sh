# OrchestratorRunCommandRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** |  | [optional] 
**env** | **Dict[str, str]** |  | [optional] 
**timeout_seconds** | **int** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_run_command_request import OrchestratorRunCommandRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorRunCommandRequest from a JSON string
orchestrator_run_command_request_instance = OrchestratorRunCommandRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorRunCommandRequest.to_json())

# convert the object into a dict
orchestrator_run_command_request_dict = orchestrator_run_command_request_instance.to_dict()
# create an instance of OrchestratorRunCommandRequest from a dict
orchestrator_run_command_request_from_dict = OrchestratorRunCommandRequest.from_dict(orchestrator_run_command_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


