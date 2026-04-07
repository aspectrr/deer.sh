# FluidRemoteInternalRestListCertificatesResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificates** | [**List[FluidRemoteInternalRestCertificateResponse]**](FluidRemoteInternalRestCertificateResponse.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_list_certificates_response import FluidRemoteInternalRestListCertificatesResponse

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestListCertificatesResponse from a JSON string
fluid_remote_internal_rest_list_certificates_response_instance = FluidRemoteInternalRestListCertificatesResponse.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestListCertificatesResponse.to_json())

# convert the object into a dict
fluid_remote_internal_rest_list_certificates_response_dict = fluid_remote_internal_rest_list_certificates_response_instance.to_dict()
# create an instance of FluidRemoteInternalRestListCertificatesResponse from a dict
fluid_remote_internal_rest_list_certificates_response_from_dict = FluidRemoteInternalRestListCertificatesResponse.from_dict(fluid_remote_internal_rest_list_certificates_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


