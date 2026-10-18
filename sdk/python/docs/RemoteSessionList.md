# RemoteSessionList

Cursor page of control sessions. `next_cursor` is absent on the last page and `total` is deliberately absent, the same as every other paginated list here: the endpoint does not count the whole set per page. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[RemoteSession]**](RemoteSession.md) |  | 
**next_cursor** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.remote_session_list import RemoteSessionList

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionList from a JSON string
remote_session_list_instance = RemoteSessionList.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionList.to_json())

# convert the object into a dict
remote_session_list_dict = remote_session_list_instance.to_dict()
# create an instance of RemoteSessionList from a dict
remote_session_list_from_dict = RemoteSessionList.from_dict(remote_session_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


