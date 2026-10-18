# OfflineAlertInput


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enabled** | **bool** |  | 
**after_minutes** | **int** | Omitting it (or sending 0) stores the default of 10. | [optional] 
**repeat_minutes** | **int** | 0 (the default) sends one alert per outage. | [optional] 
**snoozed_until** | **datetime** | Silences alerts until this instant; null clears an existing snooze. | [optional] 
**maintenance_windows** | [**List[MaintenanceWindow]**](MaintenanceWindow.md) | Recurring quiet periods evaluated in &#x60;maintenance_timezone&#x60;. | [optional] 
**maintenance_timezone** | **str** | IANA zone name for the windows; empty means the server&#39;s local zone. | [optional] 

## Example

```python
from quickdesk_signaling.models.offline_alert_input import OfflineAlertInput

# TODO update the JSON string below
json = "{}"
# create an instance of OfflineAlertInput from a JSON string
offline_alert_input_instance = OfflineAlertInput.from_json(json)
# print the JSON string representation of the object
print(OfflineAlertInput.to_json())

# convert the object into a dict
offline_alert_input_dict = offline_alert_input_instance.to_dict()
# create an instance of OfflineAlertInput from a dict
offline_alert_input_from_dict = OfflineAlertInput.from_dict(offline_alert_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


