# AccessGrant


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**device_id** | **str** |  | 
**code** | **str** | Returned only in the creation response; treat it as a secret. | 
**label** | **str** |  | [optional] 
**expires_at** | **datetime** |  | 
**max_uses** | **int** |  | 
**uses** | **int** |  | 
**active** | **bool** |  | 
**revoked_at** | **datetime** |  | [optional] 
**last_used_at** | **datetime** |  | [optional] 
**bound_client_id** | **str** |  | 
**bound_ips** | **List[str]** |  | 
**last_used_client_id** | **str** |  | [optional] 
**last_used_ip** | **str** |  | [optional] 
**created_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.access_grant import AccessGrant

# TODO update the JSON string below
json = "{}"
# create an instance of AccessGrant from a JSON string
access_grant_instance = AccessGrant.from_json(json)
# print the JSON string representation of the object
print(AccessGrant.to_json())

# convert the object into a dict
access_grant_dict = access_grant_instance.to_dict()
# create an instance of AccessGrant from a dict
access_grant_from_dict = AccessGrant.from_dict(access_grant_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


