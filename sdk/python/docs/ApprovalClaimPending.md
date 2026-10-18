# ApprovalClaimPending

The owner has not decided yet; poll again after `poll_after_ms`.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | 
**expires_at** | **datetime** |  | 
**poll_after_ms** | **int** |  | 

## Example

```python
from quickdesk_signaling.models.approval_claim_pending import ApprovalClaimPending

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalClaimPending from a JSON string
approval_claim_pending_instance = ApprovalClaimPending.from_json(json)
# print the JSON string representation of the object
print(ApprovalClaimPending.to_json())

# convert the object into a dict
approval_claim_pending_dict = approval_claim_pending_instance.to_dict()
# create an instance of ApprovalClaimPending from a dict
approval_claim_pending_from_dict = ApprovalClaimPending.from_dict(approval_claim_pending_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


