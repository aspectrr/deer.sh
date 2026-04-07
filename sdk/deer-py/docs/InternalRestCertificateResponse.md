# InternalRestCertificateResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **str** |  | [optional] 
**identity** | **str** |  | [optional] 
**is_expired** | **bool** |  | [optional] 
**issued_at** | **str** |  | [optional] 
**principals** | **List[str]** |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**serial_number** | **int** |  | [optional] 
**status** | **str** |  | [optional] 
**ttl_seconds** | **int** |  | [optional] 
**user_id** | **str** |  | [optional] 
**valid_after** | **str** |  | [optional] 
**valid_before** | **str** |  | [optional] 
**vm_id** | **str** |  | [optional] 

## Example

```python
from fluid.models.internal_rest_certificate_response import InternalRestCertificateResponse

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestCertificateResponse from a JSON string
internal_rest_certificate_response_instance = InternalRestCertificateResponse.from_json(json)
# print the JSON string representation of the object
print(InternalRestCertificateResponse.to_json())

# convert the object into a dict
internal_rest_certificate_response_dict = internal_rest_certificate_response_instance.to_dict()
# create an instance of InternalRestCertificateResponse from a dict
internal_rest_certificate_response_from_dict = InternalRestCertificateResponse.from_dict(internal_rest_certificate_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


