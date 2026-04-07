# OrchestratorRunSourceRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** |  | [optional] 
**timeout_seconds** | **int** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_run_source_request import OrchestratorRunSourceRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorRunSourceRequest from a JSON string
orchestrator_run_source_request_instance = OrchestratorRunSourceRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorRunSourceRequest.to_json())

# convert the object into a dict
orchestrator_run_source_request_dict = orchestrator_run_source_request_instance.to_dict()
# create an instance of OrchestratorRunSourceRequest from a dict
orchestrator_run_source_request_from_dict = OrchestratorRunSourceRequest.from_dict(orchestrator_run_source_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


