# FluidRemoteInternalRestDiffRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**from_snapshot** | **str** | required | [optional] 
**to_snapshot** | **str** | required | [optional] 

## Example

```python
from fluid.models.fluid_remote_internal_rest_diff_request import FluidRemoteInternalRestDiffRequest

# TODO update the JSON string below
json = "{}"
# create an instance of FluidRemoteInternalRestDiffRequest from a JSON string
fluid_remote_internal_rest_diff_request_instance = FluidRemoteInternalRestDiffRequest.from_json(json)
# print the JSON string representation of the object
print(FluidRemoteInternalRestDiffRequest.to_json())

# convert the object into a dict
fluid_remote_internal_rest_diff_request_dict = fluid_remote_internal_rest_diff_request_instance.to_dict()
# create an instance of FluidRemoteInternalRestDiffRequest from a dict
fluid_remote_internal_rest_diff_request_from_dict = FluidRemoteInternalRestDiffRequest.from_dict(fluid_remote_internal_rest_diff_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


