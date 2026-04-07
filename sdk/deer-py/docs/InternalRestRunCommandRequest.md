# InternalRestRunCommandRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** | required | [optional] 
**env** | **Dict[str, str]** | optional | [optional] 
**private_key_path** | **str** | optional; if empty, uses managed credentials (requires SSH CA) | [optional] 
**timeout_sec** | **int** | optional; default from service config | [optional] 
**user** | **str** | optional; defaults to \&quot;sandbox\&quot; when using managed credentials | [optional] 

## Example

```python
from fluid.models.internal_rest_run_command_request import InternalRestRunCommandRequest

# TODO update the JSON string below
json = "{}"
# create an instance of InternalRestRunCommandRequest from a JSON string
internal_rest_run_command_request_instance = InternalRestRunCommandRequest.from_json(json)
# print the JSON string representation of the object
print(InternalRestRunCommandRequest.to_json())

# convert the object into a dict
internal_rest_run_command_request_dict = internal_rest_run_command_request_instance.to_dict()
# create an instance of InternalRestRunCommandRequest from a dict
internal_rest_run_command_request_from_dict = InternalRestRunCommandRequest.from_dict(internal_rest_run_command_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


