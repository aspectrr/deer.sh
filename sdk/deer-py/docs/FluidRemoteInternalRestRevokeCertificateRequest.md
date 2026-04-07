# FluidRemoteInternalRestRevokeCertificateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**reason** | **str** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_revoke_certificate_request import FluidRemoteInternalRestRevokeCertificateRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestRevokeCertificateRequest from a JSON string
fluid_remote_internal_rest_revoke_certificate_request_instance = FluidRemoteInternalRestRevokeCertificateRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestRevokeCertificateRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_revoke_certificate_request_dict = fluid_remote_internal_rest_revoke_certificate_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestRevokeCertificateRequest from a dict
fluid_remote_internal_rest_revoke_certificate_request_from_dict = FluidRemoteInternalRestRevokeCertificateRequest.from_dict(fluid_remote_internal_rest_revoke_certificate_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


