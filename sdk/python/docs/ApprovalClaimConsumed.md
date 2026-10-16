# ApprovalClaimConsumed

The approval was already exchanged for a token.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | 
**message** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.approval_claim_consumed import ApprovalClaimConsumed

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalClaimConsumed from a JSON string
approval_claim_consumed_instance = ApprovalClaimConsumed.from_json(json)
# print the JSON string representation of the object
print(ApprovalClaimConsumed.to_json())

# convert the object into a dict
approval_claim_consumed_dict = approval_claim_consumed_instance.to_dict()
# create an instance of ApprovalClaimConsumed from a dict
approval_claim_consumed_from_dict = ApprovalClaimConsumed.from_dict(approval_claim_consumed_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


