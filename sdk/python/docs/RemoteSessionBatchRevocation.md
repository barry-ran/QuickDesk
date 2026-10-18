# RemoteSessionBatchRevocation

The result of clearing every live control session off one device. Unlike the single-session response this is a *report*: a batch can half succeed, so the counts and the per-session entries are the answer. `truncated` is true when the device had more live controllers than one call handles, so a caller that sees it knows the machine may not be empty yet. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 
**revoked** | **int** | How many sessions were closed and confirmed. | 
**failed** | **int** | How many were live but could not be closed. | 
**truncated** | **bool** | True when more live sessions existed than one call handles (the cap is 200). | 
**sessions** | [**List[RemoteSessionBatchRevocationSessionsInner]**](RemoteSessionBatchRevocationSessionsInner.md) | One entry per session the call attempted, newest first. | 

## Example

```python
from quickdesk_signaling.models.remote_session_batch_revocation import RemoteSessionBatchRevocation

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionBatchRevocation from a JSON string
remote_session_batch_revocation_instance = RemoteSessionBatchRevocation.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionBatchRevocation.to_json())

# convert the object into a dict
remote_session_batch_revocation_dict = remote_session_batch_revocation_instance.to_dict()
# create an instance of RemoteSessionBatchRevocation from a dict
remote_session_batch_revocation_from_dict = RemoteSessionBatchRevocation.from_dict(remote_session_batch_revocation_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


