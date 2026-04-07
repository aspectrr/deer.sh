# FluidRemoteInternalRestCaPublicKeyResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**public_key** | **str** | PublicKey is the CA public key in OpenSSH format. | [optional] 
**usage** | **str** | Usage explains how to use this key. | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_ca_public_key_response import FluidRemoteInternalRestCaPublicKeyResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestCaPublicKeyResponse from a JSON string
fluid_remote_internal_rest_ca_public_key_response_instance = FluidRemoteInternalRestCaPublicKeyResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestCaPublicKeyResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_ca_public_key_response_dict = fluid_remote_internal_rest_ca_public_key_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestCaPublicKeyResponse from a dict
fluid_remote_internal_rest_ca_public_key_response_from_dict = FluidRemoteInternalRestCaPublicKeyResponse.from_dict(fluid_remote_internal_rest_ca_public_key_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


