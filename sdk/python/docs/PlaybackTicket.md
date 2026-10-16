# PlaybackTicket

A signed, time-limited URL for one recording's media or preview. It carries no credentials, so it must be treated as a bearer token: anyone who holds the link can fetch the file until `expires_at`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**url** | **str** | Absolute path (with the signed &#x60;t&#x60; parameter) to fetch. | 
**expires_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.playback_ticket import PlaybackTicket

# TODO update the JSON string below
json = "{}"
# create an instance of PlaybackTicket from a JSON string
playback_ticket_instance = PlaybackTicket.from_json(json)
# print the JSON string representation of the object
print(PlaybackTicket.to_json())

# convert the object into a dict
playback_ticket_dict = playback_ticket_instance.to_dict()
# create an instance of PlaybackTicket from a dict
playback_ticket_from_dict = PlaybackTicket.from_dict(playback_ticket_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


