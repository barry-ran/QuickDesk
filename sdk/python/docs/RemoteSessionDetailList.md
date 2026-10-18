# RemoteSessionDetailList

Cursor page of sessions that each carry their related records (see `RemoteSessionDetail`). `next_cursor` is absent on the last page. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[RemoteSessionDetail]**](RemoteSessionDetail.md) |  | 
**next_cursor** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.remote_session_detail_list import RemoteSessionDetailList

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionDetailList from a JSON string
remote_session_detail_list_instance = RemoteSessionDetailList.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionDetailList.to_json())

# convert the object into a dict
remote_session_detail_list_dict = remote_session_detail_list_instance.to_dict()
# create an instance of RemoteSessionDetailList from a dict
remote_session_detail_list_from_dict = RemoteSessionDetailList.from_dict(remote_session_detail_list_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


