# ApprovalSummaryByStatus

Always carries all five states, zeroes included.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**pending** | **int** |  | 
**approved** | **int** |  | 
**denied** | **int** |  | 
**expired** | **int** |  | 
**consumed** | **int** |  | 

## Example

```python
from quickdesk_signaling.models.approval_summary_by_status import ApprovalSummaryByStatus

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalSummaryByStatus from a JSON string
approval_summary_by_status_instance = ApprovalSummaryByStatus.from_json(json)
# print the JSON string representation of the object
print(ApprovalSummaryByStatus.to_json())

# convert the object into a dict
approval_summary_by_status_dict = approval_summary_by_status_instance.to_dict()
# create an instance of ApprovalSummaryByStatus from a dict
approval_summary_by_status_from_dict = ApprovalSummaryByStatus.from_dict(approval_summary_by_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


