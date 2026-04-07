# InternalRestInjectSSHKeyRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**public_key** | **str** | required | [optional] 
**username** | **str** | required (explicit); typical: \&quot;ubuntu\&quot; or \&quot;centos\&quot; | [optional] 

## Example

```python
from fluid.models.internal_rest_inject_ssh_key_request import InternalRestInjectSSHKeyRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestInjectSSHKeyRequest from a JSON string
internal_rest_inject_ssh_key_request_instance = InternalRestInjectSSHKeyRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestInjectSSHKeyRequest.to_json())

# convert the object into a dict
internal_rest_inject_ssh_key_request_dict = internal_rest_inject_ssh_key_request_instance.to_dict()
# create an instance of InternalRestInjectSSHKeyRequest from a dict
internal_rest_inject_ssh_key_request_from_dict = InternalRestInjectSSHKeyRequest.from_dict(internal_rest_inject_ssh_key_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


