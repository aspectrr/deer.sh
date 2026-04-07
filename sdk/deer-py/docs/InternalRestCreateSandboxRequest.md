# InternalRestCreateSandboxRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_id** | **str** | required | [optional] 
**auto_start** | **bool** | optional; if true, start the VM immediately after creation | [optional] 
**cpu** | **int** | optional; default from service config if &lt;&#x3D;0 | [optional] 
**memory_mb** | **int** | optional; default from service config if &lt;&#x3D;0 | [optional] 
**source_vm_name** | **str** | required; name of existing VM in libvirt to clone from | [optional] 
**ttl_seconds** | **int** | optional; TTL for auto garbage collection | [optional] 
**wait_for_ip** | **bool** | optional; if true and auto_start, wait for IP discovery | [optional] 

## Example

```python
from fluid.models.internal_rest_create_sandbox_request import InternalRestCreateSandboxRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestCreateSandboxRequest from a JSON string
internal_rest_create_sandbox_request_instance = InternalRestCreateSandboxRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestCreateSandboxRequest.to_json())

# convert the object into a dict
internal_rest_create_sandbox_request_dict = internal_rest_create_sandbox_request_instance.to_dict()
# create an instance of InternalRestCreateSandboxRequest from a dict
internal_rest_create_sandbox_request_from_dict = InternalRestCreateSandboxRequest.from_dict(internal_rest_create_sandbox_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


