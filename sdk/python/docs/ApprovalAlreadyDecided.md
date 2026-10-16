# ApprovalAlreadyDecided

Returned by approve/deny when the request was already decided (or expired) so the click is reported honestly instead of silently succeeding. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | 
**message** | **str** |  | 

## Example

```python
from quickdesk_signaling.models.approval_already_decided import ApprovalAlreadyDecided

# TODO update the JSON string below
json = "{}"
# create an instance of ApprovalAlreadyDecided from a JSON string
approval_already_decided_instance = ApprovalAlreadyDecided.from_json(json)
# print the JSON string representation of the object
print(ApprovalAlreadyDecided.to_json())

# convert the object into a dict
approval_already_decided_dict = approval_already_decided_instance.to_dict()
# create an instance of ApprovalAlreadyDecided from a dict
approval_already_decided_from_dict = ApprovalAlreadyDecided.from_dict(approval_already_decided_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


