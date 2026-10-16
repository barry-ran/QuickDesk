# SessionRecordingList

Cursor page of recordings. `next_cursor` is absent on the last page; `total` is intentionally absent because this endpoint does not count the whole set per page. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[SessionRecording]**](SessionRecording.md) |  | 
**next_cursor** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.session_recording_list import SessionRecordingList

# TODO update the JSON string below
json = "{}"
# create an instance of SessionRecordingList from a JSON string
session_recording_list_instance = SessionRecordingList.from_json(json)
# print the JSON string representation of the object
print(SessionRecordingList.to_json())

# convert the object into a dict
session_recording_list_dict = session_recording_list_instance.to_dict()
# create an instance of SessionRecordingList from a dict
session_recording_list_from_dict = SessionRecordingList.from_dict(session_recording_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


