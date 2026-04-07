# FluidRemoteInternalRestInjectSSHKeyRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**public_key** | **str** | required | [optional] 
**username** | **str** | required (explicit); typical: \&quot;ubuntu\&quot; or \&quot;centos\&quot; | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_inject_ssh_key_request import FluidRemoteInternalRestInjectSSHKeyRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestInjectSSHKeyRequest from a JSON string
fluid_remote_internal_rest_inject_ssh_key_request_instance = FluidRemoteInternalRestInjectSSHKeyRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestInjectSSHKeyRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_inject_ssh_key_request_dict = fluid_remote_internal_rest_inject_ssh_key_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestInjectSSHKeyRequest from a dict
fluid_remote_internal_rest_inject_ssh_key_request_from_dict = FluidRemoteInternalRestInjectSSHKeyRequest.from_dict(fluid_remote_internal_rest_inject_ssh_key_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


