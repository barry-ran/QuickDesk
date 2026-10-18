# ApprovalClaimRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**approval_id** | **int** |  | 
**secret** | **str** | The &#x60;approval_secret&#x60; returned once by &#x60;access-code:verify&#x60;. | 

## Example

```python
from quickdesk_signaling.models.approval_claim_request import ApprovalClaimRequest

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalClaimRequest from a JSON string
approval_claim_request_instance = ApprovalClaimRequest.from_json(json)
# print the JSON string representation of the object
print(ApprovalClaimRequest.to_json())

# convert the object into a dict
approval_claim_request_dict = approval_claim_request_instance.to_dict()
# create an instance of ApprovalClaimRequest from a dict
approval_claim_request_from_dict = ApprovalClaimRequest.from_dict(approval_claim_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


