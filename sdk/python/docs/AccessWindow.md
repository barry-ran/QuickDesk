# AccessWindow

`start > end` wraps past midnight and the day filter then applies to the day the window starts; `start == end` means all day on the selected days; an empty `days` list means every day. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**days** | **List[int]** |  | [optional] 
**start** | **str** |  | 
**end** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.access_window import AccessWindow

# TODO update the JSON string below
json = "{}"
# create an instance of AccessWindow from a JSON string
access_window_instance = AccessWindow.from_json(json)
# print the JSON string representation of the object
print(AccessWindow.to_json())

# convert the object into a dict
access_window_dict = access_window_instance.to_dict()
# create an instance of AccessWindow from a dict
access_window_from_dict = AccessWindow.from_dict(access_window_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


