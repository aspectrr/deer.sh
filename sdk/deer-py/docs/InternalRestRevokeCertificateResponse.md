# InternalRestRevokeCertificateResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | [optional] 
**message** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_revoke_certificate_response import InternalRestRevokeCertificateResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestRevokeCertificateResponse from a JSON string
internal_rest_revoke_certificate_response_instance = InternalRestRevokeCertificateResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestRevokeCertificateResponse.to_json())

# convert the object into a dict
internal_rest_revoke_certificate_response_dict = internal_rest_revoke_certificate_response_instance.to_dict()
# create an instance of InternalRestRevokeCertificateResponse from a dict
internal_rest_revoke_certificate_response_from_dict = InternalRestRevokeCertificateResponse.from_dict(internal_rest_revoke_certificate_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


