# OfflineAlertRule


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 
**enabled** | **bool** |  | 
**after_minutes** | **int** |  | 
**repeat_minutes** | **int** |  | 
**snoozed_until** | **datetime** |  | [optional] 
**maintenance_windows** | [**List[MaintenanceWindow]**](MaintenanceWindow.md) |  | 
**maintenance_timezone** | **str** |  | [optional] 
**in_maintenance_window** | **bool** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. | 
**offline_since** | **datetime** | The evaluator&#39;s episode state; present only while the device is in an outage. | [optional] 
**last_alerted_at** | **datetime** |  | [optional] 
**updated_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.offline_alert_rule import OfflineAlertRule

# TODO update the JSON string below
json = "{}"
# create an instance of OfflineAlertRule from a JSON string
offline_alert_rule_instance = OfflineAlertRule.from_json(json)
# print the JSON string representation of the object
print(OfflineAlertRule.to_json())

# convert the object into a dict
offline_alert_rule_dict = offline_alert_rule_instance.to_dict()
# create an instance of OfflineAlertRule from a dict
offline_alert_rule_from_dict = OfflineAlertRule.from_dict(offline_alert_rule_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


