# ApprovalSummaryDeniedSourcesInner


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**client_ip** | **str** |  | 
**denials** | **int** |  | 
**last_denied_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.approval_summary_denied_sources_inner import ApprovalSummaryDeniedSourcesInner

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalSummaryDeniedSourcesInner from a JSON string
approval_summary_denied_sources_inner_instance = ApprovalSummaryDeniedSourcesInner.from_json(json)
# print the JSON string representation of the object
print(ApprovalSummaryDeniedSourcesInner.to_json())

# convert the object into a dict
approval_summary_denied_sources_inner_dict = approval_summary_denied_sources_inner_instance.to_dict()
# create an instance of ApprovalSummaryDeniedSourcesInner from a dict
approval_summary_denied_sources_inner_from_dict = ApprovalSummaryDeniedSourcesInner.from_dict(approval_summary_denied_sources_inner_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


