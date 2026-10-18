# RemoteSessionDetailCounts


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**recordings** | **int** |  | 
**access_log** | **int** |  | 

## Example

```python
from quickdesk_signaling.models.remote_session_detail_counts import RemoteSessionDetailCounts

# TODO update the JSON string below
json = "{}"
# create an instance of RemoteSessionDetailCounts from a JSON string
remote_session_detail_counts_instance = RemoteSessionDetailCounts.from_json(json)
# print the JSON string representation of the object
print(RemoteSessionDetailCounts.to_json())

# convert the object into a dict
remote_session_detail_counts_dict = remote_session_detail_counts_instance.to_dict()
# create an instance of RemoteSessionDetailCounts from a dict
remote_session_detail_counts_from_dict = RemoteSessionDetailCounts.from_dict(remote_session_detail_counts_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


