# RemoteSessionRevocation

The result of ending one live control session. `revoked` is always `true` - the endpoint answers `409` rather than returning a false here, so a caller never has to interpret a success response as \"maybe\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**session_id** | **str** |  | 
**device_id** | **str** |  | 
**revoked** | **bool** |  | 

## Example

```python
from quickdesk_signaling.models.remote_session_revocation import RemoteSessionRevocation

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionRevocation from a JSON string
remote_session_revocation_instance = RemoteSessionRevocation.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionRevocation.to_json())

# convert the object into a dict
remote_session_revocation_dict = remote_session_revocation_instance.to_dict()
# create an instance of RemoteSessionRevocation from a dict
remote_session_revocation_from_dict = RemoteSessionRevocation.from_dict(remote_session_revocation_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


