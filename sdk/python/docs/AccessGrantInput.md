# AccessGrantInput


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**label** | **str** |  | [optional] 
**expires_in_minutes** | **int** |  | 
**max_uses** | **int** | 0 means \&quot;no use limit\&quot;; the grant still expires. | [optional] 
**code** | **str** | Optional caller-chosen code; omitted means the server generates one. | [optional] 
**bound_client_id** | **str** | Self-reported controller identity the code is bound to. | [optional] 
**bound_ips** | **List[str]** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.access_grant_input import AccessGrantInput

# TODO update the JSON string below
json = "{}"
# create an instance of AccessGrantInput from a JSON string
access_grant_input_instance = AccessGrantInput.from_json(json)
# print the JSON string representation of the object
print(AccessGrantInput.to_json())

# convert the object into a dict
access_grant_input_dict = access_grant_input_instance.to_dict()
# create an instance of AccessGrantInput from a dict
access_grant_input_from_dict = AccessGrantInput.from_dict(access_grant_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


