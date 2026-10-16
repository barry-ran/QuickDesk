# ApprovalSummary


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**since** | **datetime** | Inclusive start of the window the counts cover. | 
**total** | **int** | Every request created inside the window, whatever its outcome. | 
**by_status** | [**ApprovalSummaryByStatus**](ApprovalSummaryByStatus.md) |  | 
**denied_sources** | [**List[ApprovalSummaryDeniedSourcesInner]**](ApprovalSummaryDeniedSourcesInner.md) | Refused sources inside the window, most refusals first. | 
**denial_cooldown_seconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_SECONDS, so an owner can tell \&quot;refused right now\&quot; from \&quot;refused a while ago\&quot;. | 
**denial_cooldown_multiplier** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MULTIPLIER; 1 means each further consecutive refusal costs the same as the last. | 
**denial_cooldown_max_seconds** | **int** | The live APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS, the ceiling the escalation cannot pass; 0 means uncapped. | 

## Example

```python
from quickdesk_signaling.models.approval_summary import ApprovalSummary

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalSummary from a JSON string
approval_summary_instance = ApprovalSummary.from_json(json)
# print the JSON string representation of the object
print(ApprovalSummary.to_json())

# convert the object into a dict
approval_summary_dict = approval_summary_instance.to_dict()
# create an instance of ApprovalSummary from a dict
approval_summary_from_dict = ApprovalSummary.from_dict(approval_summary_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


