# FluidRemoteInternalRestRequestAccessRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**public_key** | **str** | PublicKey is the user&#39;s SSH public key in OpenSSH format. | [optional] 
**sandbox_id** | **str** | SandboxID is the target sandbox. | [optional] 
**ttl_minutes** | **int** | TTLMinutes is the requested access duration (1-10 minutes). | [optional] 
**user_id** | **str** | UserID identifies the requesting user. | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_request_access_request import FluidRemoteInternalRestRequestAccessRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestRequestAccessRequest from a JSON string
fluid_remote_internal_rest_request_access_request_instance = FluidRemoteInternalRestRequestAccessRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestRequestAccessRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_request_access_request_dict = fluid_remote_internal_rest_request_access_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestRequestAccessRequest from a dict
fluid_remote_internal_rest_request_access_request_from_dict = FluidRemoteInternalRestRequestAccessRequest.from_dict(fluid_remote_internal_rest_request_access_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


