# ApproveAccessApprovalRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**reason** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.approve_access_approval_request import ApproveAccessApprovalRequest

# TODO update the JSON string below
json = "{}"
# create an instance of ApproveAccessApprovalRequest from a JSON string
approve_access_approval_request_instance = ApproveAccessApprovalRequest.from_json(json)
# print the JSON string representation of the object
print(ApproveAccessApprovalRequest.to_json())

# convert the object into a dict
approve_access_approval_request_dict = approve_access_approval_request_instance.to_dict()
# create an instance of ApproveAccessApprovalRequest from a dict
approve_access_approval_request_from_dict = ApproveAccessApprovalRequest.from_dict(approve_access_approval_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


