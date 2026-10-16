# AuthSessionRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**identifier** | **str** | Username, phone, or email, depending on how the account was created. | 
**password** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.auth_session_request import AuthSessionRequest

# TODO update the JSON string below
json = "{}"
# create an instance of AuthSessionRequest from a JSON string
auth_session_request_instance = AuthSessionRequest.from_json(json)
# print the JSON string representation of the object
print(AuthSessionRequest.to_json())

# convert the object into a dict
auth_session_request_dict = auth_session_request_instance.to_dict()
# create an instance of AuthSessionRequest from a dict
auth_session_request_from_dict = AuthSessionRequest.from_dict(auth_session_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


