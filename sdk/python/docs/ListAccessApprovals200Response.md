# ListAccessApprovals200Response


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[AccessApproval]**](AccessApproval.md) |  | 

## Example

```python
from quickdesk_signaling.models.list_access_approvals200_response import ListAccessApprovals200Response

# TODO update the JSON string below
json = "{}"
# create an instance of ListAccessApprovals200Response from a JSON string
list_access_approvals200_response_instance = ListAccessApprovals200Response.from_json(json)
# print the JSON string representation of the object
print(ListAccessApprovals200Response.to_json())

# convert the object into a dict
list_access_approvals200_response_dict = list_access_approvals200_response_instance.to_dict()
# create an instance of ListAccessApprovals200Response from a dict
list_access_approvals200_response_from_dict = ListAccessApprovals200Response.from_dict(list_access_approvals200_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


