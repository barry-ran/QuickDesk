# UpdateCheckResult


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**update_available** | **bool** |  | 
**mandatory** | **bool** |  | 
**channel** | **str** |  | 
**platform** | **str** |  | 
**arch** | **str** |  | 
**current_version** | **str** | Echoed when the caller supplied it. | [optional] 
**latest** | [**UpdateRelease**](UpdateRelease.md) |  | 

## Example

```python
from quickdesk_signaling.models.update_check_result import UpdateCheckResult

# TODO update the JSON string below
json = "{}"
# create an instance of UpdateCheckResult from a JSON string
update_check_result_instance = UpdateCheckResult.from_json(json)
# print the JSON string representation of the object
print(UpdateCheckResult.to_json())

# convert the object into a dict
update_check_result_dict = update_check_result_instance.to_dict()
# create an instance of UpdateCheckResult from a dict
update_check_result_from_dict = UpdateCheckResult.from_dict(update_check_result_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


