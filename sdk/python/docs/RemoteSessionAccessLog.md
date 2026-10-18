# RemoteSessionAccessLog

One device access-log row that carries a session id - in practice the `session_started` entry the server wrote when the controller authenticated, which is what pairs \"an attempt\" with \"the session it became\". The full timeline is documented separately in `docs/user-api-docs.md` (§5.3.1); this is the part a session owns. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**device_id** | **str** |  | 
**owner_id** | **int** | The device&#39;s owner at the time of the attempt; 0 when unbound. | 
**client_id** | **str** |  | 
**session_id** | **str** |  | 
**ip** | **str** |  | 
**result** | **str** | The canonical access result, e.g. &#x60;session_started&#x60;. | 
**reason** | **str** |  | 
**created_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.remote_session_access_log import RemoteSessionAccessLog

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionAccessLog from a JSON string
remote_session_access_log_instance = RemoteSessionAccessLog.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionAccessLog.to_json())

# convert the object into a dict
remote_session_access_log_dict = remote_session_access_log_instance.to_dict()
# create an instance of RemoteSessionAccessLog from a dict
remote_session_access_log_from_dict = RemoteSessionAccessLog.from_dict(remote_session_access_log_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


