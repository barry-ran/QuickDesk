# NotificationChannel


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**user_id** | **int** |  | 
**name** | **str** |  | 
**url** | **str** |  | 
**events** | **str** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;.  | 
**enabled** | **bool** |  | 
**last_triggered** | **datetime** |  | [optional] 
**last_status** | **int** | HTTP status of the most recent delivery attempt; 0 means none yet. | 
**created_at** | **datetime** |  | [optional] 
**updated_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.notification_channel import NotificationChannel

# TODO update the JSON string below
json = "{}"
# create an instance of NotificationChannel from a JSON string
notification_channel_instance = NotificationChannel.from_json(json)
# print the JSON string representation of the object
print(NotificationChannel.to_json())

# convert the object into a dict
notification_channel_dict = notification_channel_instance.to_dict()
# create an instance of NotificationChannel from a dict
notification_channel_from_dict = NotificationChannel.from_dict(notification_channel_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


