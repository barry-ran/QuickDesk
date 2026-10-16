# RemoteSessionBatchRevocationSessionsInner


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session_id** | **str** |  | 
**revoked** | **bool** |  | 
**reason** | **str** | Why the session could not be closed; present only when &#x60;revoked&#x60; is false. | [optional] 

## Example

```python
from quickdesk_signaling.models.remote_session_batch_revocation_sessions_inner import RemoteSessionBatchRevocationSessionsInner

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionBatchRevocationSessionsInner from a JSON string
remote_session_batch_revocation_sessions_inner_instance = RemoteSessionBatchRevocationSessionsInner.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionBatchRevocationSessionsInner.to_json())

# convert the object into a dict
remote_session_batch_revocation_sessions_inner_dict = remote_session_batch_revocation_sessions_inner_instance.to_dict()
# create an instance of RemoteSessionBatchRevocationSessionsInner from a dict
remote_session_batch_revocation_sessions_inner_from_dict = RemoteSessionBatchRevocationSessionsInner.from_dict(remote_session_batch_revocation_sessions_inner_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


