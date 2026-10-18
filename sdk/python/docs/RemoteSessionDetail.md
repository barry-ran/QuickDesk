# RemoteSessionDetail

One control session with the records that share its `session_id`. Both lists are capped at 100 entries and present-but-empty when there is nothing to show, so a client never has to distinguish \"no recordings\" from \"field missing\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session** | [**RemoteSession**](RemoteSession.md) |  | 
**recordings** | [**List[SessionRecording]**](SessionRecording.md) |  | 
**access_log** | [**List[RemoteSessionAccessLog]**](RemoteSessionAccessLog.md) |  | 
**counts** | [**RemoteSessionDetailCounts**](RemoteSessionDetailCounts.md) |  | 

## Example

```python
from quickdesk_signaling.models.remote_session_detail import RemoteSessionDetail

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionDetail from a JSON string
remote_session_detail_instance = RemoteSessionDetail.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionDetail.to_json())

# convert the object into a dict
remote_session_detail_dict = remote_session_detail_instance.to_dict()
# create an instance of RemoteSessionDetail from a dict
remote_session_detail_from_dict = RemoteSessionDetail.from_dict(remote_session_detail_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


