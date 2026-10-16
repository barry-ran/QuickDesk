# RemoteSession

One controller holding, or having held, a signaling session against a device. `active` and `duration_seconds` are derived: the first from `ended_at`, the second is the run length so far while the session is still open, so a client never has to pick its own notion of \"now\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**session_id** | **str** | The signaling session id the server assigned; unique per session. | 
**user_id** | **int** | The device&#39;s owner. 0 when the device was unbound at the time. | 
**device_id** | **str** |  | 
**client_id** | **str** | The controller-declared client id, the same value the device access log records. | 
**client_ip** | **str** |  | 
**started_at** | **datetime** |  | 
**ended_at** | **datetime** |  | 
**end_reason** | **str** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away, &#x60;server_restart&#x60; for rows a restart found still open. | 
**active** | **bool** |  | 
**duration_seconds** | **int** |  | 
**created_at** | **datetime** |  | 
**updated_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.remote_session import RemoteSession

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSession from a JSON string
remote_session_instance = RemoteSession.from_json(json)
# print the JSON string representation of the object
print(RemoteSession.to_json())

# convert the object into a dict
remote_session_dict = remote_session_instance.to_dict()
# create an instance of RemoteSession from a dict
remote_session_from_dict = RemoteSession.from_dict(remote_session_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


