# OrchestratorReadSourceRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **str** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_read_source_request import OrchestratorReadSourceRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorReadSourceRequest from a JSON string
orchestrator_read_source_request_instance = OrchestratorReadSourceRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorReadSourceRequest.to_json())

# convert the object into a dict
orchestrator_read_source_request_dict = orchestrator_read_source_request_instance.to_dict()
# create an instance of OrchestratorReadSourceRequest from a dict
orchestrator_read_source_request_from_dict = OrchestratorReadSourceRequest.from_dict(orchestrator_read_source_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


