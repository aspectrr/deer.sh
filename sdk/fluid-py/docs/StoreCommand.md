# StoreCommand


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **str** |  | [optional] 
**duration_ms** | **int** |  | [optional] 
**ended_at** | **str** |  | [optional] 
**exit_code** | **int** |  | [optional] 
**id** | **str** |  | [optional] 
**sandbox_id** | **str** |  | [optional] 
**started_at** | **str** |  | [optional] 
**stderr** | **str** |  | [optional] 
**stdout** | **str** |  | [optional] 

## Example

```python
from fluid.models.store_command import StoreCommand

# TODO update the JSON string below
json = "{}"
# create an instance of StoreCommand from a JSON string
store_command_instance = StoreCommand.from_json(json)
# print the JSON string representation of the object
print(StoreCommand.to_json())

# convert the object into a dict
store_command_dict = store_command_instance.to_dict()
# create an instance of StoreCommand from a dict
store_command_from_dict = StoreCommand.from_dict(store_command_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


