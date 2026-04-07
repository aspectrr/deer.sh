# InternalRestRevokeCertificateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**reason** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_revoke_certificate_request import InternalRestRevokeCertificateRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestRevokeCertificateRequest from a JSON string
internal_rest_revoke_certificate_request_instance = InternalRestRevokeCertificateRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestRevokeCertificateRequest.to_json())

# convert the object into a dict
internal_rest_revoke_certificate_request_dict = internal_rest_revoke_certificate_request_instance.to_dict()
# create an instance of InternalRestRevokeCertificateRequest from a dict
internal_rest_revoke_certificate_request_from_dict = InternalRestRevokeCertificateRequest.from_dict(internal_rest_revoke_certificate_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


