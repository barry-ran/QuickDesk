# SessionRecording

One indexed recording. The row is the control-plane metadata; the media artifact is produced by the control engine and fetched through the artifact routes. `artifact_path` is a server-side path and is not a download URL - clients should call `.../artifact-ticket` and use the URL they get back. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**user_id** | **int** |  | 
**device_id** | **str** |  | 
**device_name** | **str** |  | 
**session_id** | **str** |  | 
**status** | **str** | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker.  | 
**format** | **str** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. | 
**artifact_path** | **str** | Server-side stored path; not a client-fetchable URL. | 
**artifact_url** | **str** |  | 
**thumbnail_path** | **str** |  | 
**thumbnail_url** | **str** |  | 
**duration_seconds** | **int** |  | 
**size_bytes** | **int** |  | 
**started_at** | **datetime** |  | 
**ended_at** | **datetime** |  | [optional] 
**created_at** | **datetime** |  | 
**updated_at** | **datetime** |  | 
**user** | **object** | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present.  | [optional] 

## Example

```python
from quickdesk_signaling.models.session_recording import SessionRecording

# TODO update the JSON string below
json = "{}"
# create an instance of SessionRecording from a JSON string
session_recording_instance = SessionRecording.from_json(json)
# print the JSON string representation of the object
print(SessionRecording.to_json())

# convert the object into a dict
session_recording_dict = session_recording_instance.to_dict()
# create an instance of SessionRecording from a dict
session_recording_from_dict = SessionRecording.from_dict(session_recording_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


