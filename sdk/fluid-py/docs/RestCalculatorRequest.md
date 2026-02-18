# RestCalculatorRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_hosts** | **int** |  | [optional] 
**concurrent_sandboxes** | **int** |  | [optional] 
**hours_per_month** | **float** |  | [optional] 
**source_vms** | **int** |  | [optional] 

## Example

```python
from fluid.models.rest_calculator_request import RestCalculatorRequest

# TODO update the JSON string below
json = "{}"
# create an instance of RestCalculatorRequest from a JSON string
rest_calculator_request_instance = RestCalculatorRequest.from_json(json)
# print the JSON string representation of the object
print(RestCalculatorRequest.to_json())

# convert the object into a dict
rest_calculator_request_dict = rest_calculator_request_instance.to_dict()
# create an instance of RestCalculatorRequest from a dict
rest_calculator_request_from_dict = RestCalculatorRequest.from_dict(rest_calculator_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


