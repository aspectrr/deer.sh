# OrchestratorPrepareRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ssh_key_path** | **str** |  | [optional] 
**ssh_user** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_prepare_request import OrchestratorPrepareRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorPrepareRequest from a JSON string
orchestrator_prepare_request_instance = OrchestratorPrepareRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorPrepareRequest.to_json())

# convert the object into a dict
orchestrator_prepare_request_dict = orchestrator_prepare_request_instance.to_dict()
# create an instance of OrchestratorPrepareRequest from a dict
orchestrator_prepare_request_from_dict = OrchestratorPrepareRequest.from_dict(orchestrator_prepare_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


