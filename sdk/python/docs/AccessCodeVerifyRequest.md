# AccessCodeVerifyRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**code** | **str** |  | 
**client_id** | **str** | Optional self-reported client identity, recorded on the access log. | [optional] 

## Example

```python
from quickdesk_signaling.models.access_code_verify_request import AccessCodeVerifyRequest

# TODO update the JSON string below
json = "{}"
# create an instance of AccessCodeVerifyRequest from a JSON string
access_code_verify_request_instance = AccessCodeVerifyRequest.from_json(json)
# print the JSON string representation of the object
print(AccessCodeVerifyRequest.to_json())

# convert the object into a dict
access_code_verify_request_dict = access_code_verify_request_instance.to_dict()
# create an instance of AccessCodeVerifyRequest from a dict
access_code_verify_request_from_dict = AccessCodeVerifyRequest.from_dict(access_code_verify_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


