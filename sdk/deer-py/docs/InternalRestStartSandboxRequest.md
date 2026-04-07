# InternalRestStartSandboxRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**wait_for_ip** | **bool** | optional; default false | [optional] 

## Example

```python
from fluid.models.internal_rest_start_sandbox_request import InternalRestStartSandboxRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestStartSandboxRequest from a JSON string
internal_rest_start_sandbox_request_instance = InternalRestStartSandboxRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestStartSandboxRequest.to_json())

# convert the object into a dict
internal_rest_start_sandbox_request_dict = internal_rest_start_sandbox_request_instance.to_dict()
# create an instance of InternalRestStartSandboxRequest from a dict
internal_rest_start_sandbox_request_from_dict = InternalRestStartSandboxRequest.from_dict(internal_rest_start_sandbox_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


