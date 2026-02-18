# OrchestratorCreateSandboxRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_id** | **str** |  | [optional] 
**base_image** | **str** |  | [optional] 
**memory_mb** | **int** |  | [optional] 
**name** | **str** |  | [optional] 
**network** | **str** |  | [optional] 
**org_id** | **str** |  | [optional] 
**source_vm** | **str** |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 
**vcpus** | **int** |  | [optional] 

## Example

```python
from fluid.models.orchestrator_create_sandbox_request import OrchestratorCreateSandboxRequest

# TODO update the JSON string below
json = "{}"
# create an instance of OrchestratorCreateSandboxRequest from a JSON string
orchestrator_create_sandbox_request_instance = OrchestratorCreateSandboxRequest.from_json(json)
# print the JSON string representation of the object
print(OrchestratorCreateSandboxRequest.to_json())

# convert the object into a dict
orchestrator_create_sandbox_request_dict = orchestrator_create_sandbox_request_instance.to_dict()
# create an instance of OrchestratorCreateSandboxRequest from a dict
orchestrator_create_sandbox_request_from_dict = OrchestratorCreateSandboxRequest.from_dict(orchestrator_create_sandbox_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


