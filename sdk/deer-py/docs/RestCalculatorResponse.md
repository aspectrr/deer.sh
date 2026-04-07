# RestCalculatorResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_host_cost** | **float** |  | [optional] 
**currency** | **str** |  | [optional] 
**sandbox_cost** | **float** |  | [optional] 
**source_vm_cost** | **float** |  | [optional] 
**total_monthly** | **float** |  | [optional] 

## Example

```python
from fluid.models.rest_calculator_response import RestCalculatorResponse

# TODO update the JSON string below
json = "{}"
# create an instance of RestCalculatorResponse from a JSON string
rest_calculator_response_instance = RestCalculatorResponse.from_json(json)
# print the JSON string representation of the object
print(RestCalculatorResponse.to_json())

# convert the object into a dict
rest_calculator_response_dict = rest_calculator_response_instance.to_dict()
# create an instance of RestCalculatorResponse from a dict
rest_calculator_response_from_dict = RestCalculatorResponse.from_dict(rest_calculator_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


