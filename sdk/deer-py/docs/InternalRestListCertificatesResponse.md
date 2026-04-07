# InternalRestListCertificatesResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificates** | [**List[InternalRestCertificateResponse]**](InternalRestCertificateResponse.md) |  | [optional] 
**total** | **int** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_list_certificates_response import InternalRestListCertificatesResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestListCertificatesResponse from a JSON string
internal_rest_list_certificates_response_instance = InternalRestListCertificatesResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestListCertificatesResponse.to_json())

# convert the object into a dict
internal_rest_list_certificates_response_dict = internal_rest_list_certificates_response_instance.to_dict()
# create an instance of InternalRestListCertificatesResponse from a dict
internal_rest_list_certificates_response_from_dict = InternalRestListCertificatesResponse.from_dict(internal_rest_list_certificates_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


