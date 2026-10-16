# AccessCodeApprovalPending

The HTTP 202 body of `access-code:verify` - the same fields as `AccessCodeVerifyResponse` with `status: pending`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**approval_id** | **int** |  | 
**approval_secret** | **str** | Shown once; exchanged for a signal token at the claim endpoint. | 
**status** | **str** |  | 
**expires_at** | **datetime** |  | 
**poll_after_ms** | **int** |  | 

## Example

```python
from quickdesk_signaling.models.access_code_approval_pending import AccessCodeApprovalPending

# TODO update the JSON string below
json = "{}"
# create an instance of AccessCodeApprovalPending from a JSON string
access_code_approval_pending_instance = AccessCodeApprovalPending.from_json(json)
# print the JSON string representation of the object
print(AccessCodeApprovalPending.to_json())

# convert the object into a dict
access_code_approval_pending_dict = access_code_approval_pending_instance.to_dict()
# create an instance of AccessCodeApprovalPending from a dict
access_code_approval_pending_from_dict = AccessCodeApprovalPending.from_dict(access_code_approval_pending_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


