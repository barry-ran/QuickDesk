# AccessApproval


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**device_id** | **str** |  | 
**client_id** | **str** | Self-reported by the requesting controller. | 
**client_ip** | **str** | The address the server observed (trusted-proxy aware). | 
**source** | **str** |  | 
**grant_id** | **int** |  | [optional] 
**status** | **str** |  | 
**expires_at** | **datetime** |  | 
**decided_at** | **datetime** |  | [optional] 
**reason** | **str** |  | [optional] 
**created_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.access_approval import AccessApproval

# TODO update the JSON string below
json = "{}"
# create an instance of AccessApproval from a JSON string
access_approval_instance = AccessApproval.from_json(json)
# print the JSON string representation of the object
print(AccessApproval.to_json())

# convert the object into a dict
access_approval_dict = access_approval_instance.to_dict()
# create an instance of AccessApproval from a dict
access_approval_from_dict = AccessApproval.from_dict(access_approval_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


