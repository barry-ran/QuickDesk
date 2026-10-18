# AuthSessionResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**user** | [**User**](User.md) |  | 
**access_token** | **str** |  | 
**access_expires_at** | **datetime** |  | 
**refresh_token** | **str** |  | 
**refresh_expires_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.auth_session_response import AuthSessionResponse

# TODO update the JSON string below
json = "{}"
# create an instance of AuthSessionResponse from a JSON string
auth_session_response_instance = AuthSessionResponse.from_json(json)
# print the JSON string representation of the object
print(AuthSessionResponse.to_json())

# convert the object into a dict
auth_session_response_dict = auth_session_response_instance.to_dict()
# create an instance of AuthSessionResponse from a dict
auth_session_response_from_dict = AuthSessionResponse.from_dict(auth_session_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


