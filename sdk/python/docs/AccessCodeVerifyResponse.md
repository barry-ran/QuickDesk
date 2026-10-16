# AccessCodeVerifyResponse

The answer to `access-code:verify` and to the approval claim endpoint. `status` says which shape it is: `granted` (HTTP 200) carries `signal_token`, `pending` (HTTP 202) carries the approval the controller polls. The approval fields live on this schema rather than only on the 202 schema because a generated client declares one success type per operation - a Go caller receives the 202 body as this type, and without these properties it could not read the approval it is supposed to poll. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** | Which shape this response is. | 
**signal_token** | **str** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. | [optional] 
**expires_at** | **datetime** | When the signal token (granted) or the approval (pending) expires. | [optional] 
**approval_id** | **int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 
**approval_secret** | **str** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint.  | [optional] 
**poll_after_ms** | **int** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 

## Example

```python
from quickdesk_signaling.models.access_code_verify_response import AccessCodeVerifyResponse

# TODO update the JSON string below
json = "{}"
# create an instance of AccessCodeVerifyResponse from a JSON string
access_code_verify_response_instance = AccessCodeVerifyResponse.from_json(json)
# print the JSON string representation of the object
print(AccessCodeVerifyResponse.to_json())

# convert the object into a dict
access_code_verify_response_dict = access_code_verify_response_instance.to_dict()
# create an instance of AccessCodeVerifyResponse from a dict
access_code_verify_response_from_dict = AccessCodeVerifyResponse.from_dict(access_code_verify_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


