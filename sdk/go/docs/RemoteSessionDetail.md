# RemoteSessionDetail

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Session** | [**RemoteSession**](RemoteSession.md) |  | 
**Recordings** | [**[]SessionRecording**](SessionRecording.md) |  | 
**AccessLog** | [**[]RemoteSessionAccessLog**](RemoteSessionAccessLog.md) |  | 
**Counts** | [**RemoteSessionDetailCounts**](RemoteSessionDetailCounts.md) |  | 

## Methods

### NewRemoteSessionDetail

`func NewRemoteSessionDetail(session RemoteSession, recordings []SessionRecording, accessLog []RemoteSessionAccessLog, counts RemoteSessionDetailCounts, ) *RemoteSessionDetail`

NewRemoteSessionDetail instantiates a new RemoteSessionDetail object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionDetailWithDefaults

`func NewRemoteSessionDetailWithDefaults() *RemoteSessionDetail`

NewRemoteSessionDetailWithDefaults instantiates a new RemoteSessionDetail object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSession

`func (o *RemoteSessionDetail) GetSession() RemoteSession`

GetSession returns the Session field if non-nil, zero value otherwise.

### GetSessionOk

`func (o *RemoteSessionDetail) GetSessionOk() (*RemoteSession, bool)`

GetSessionOk returns a tuple with the Session field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSession

`func (o *RemoteSessionDetail) SetSession(v RemoteSession)`

SetSession sets Session field to given value.


### GetRecordings

`func (o *RemoteSessionDetail) GetRecordings() []SessionRecording`

GetRecordings returns the Recordings field if non-nil, zero value otherwise.

### GetRecordingsOk

`func (o *RemoteSessionDetail) GetRecordingsOk() (*[]SessionRecording, bool)`

GetRecordingsOk returns a tuple with the Recordings field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRecordings

`func (o *RemoteSessionDetail) SetRecordings(v []SessionRecording)`

SetRecordings sets Recordings field to given value.


### GetAccessLog

`func (o *RemoteSessionDetail) GetAccessLog() []RemoteSessionAccessLog`

GetAccessLog returns the AccessLog field if non-nil, zero value otherwise.

### GetAccessLogOk

`func (o *RemoteSessionDetail) GetAccessLogOk() (*[]RemoteSessionAccessLog, bool)`

GetAccessLogOk returns a tuple with the AccessLog field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessLog

`func (o *RemoteSessionDetail) SetAccessLog(v []RemoteSessionAccessLog)`

SetAccessLog sets AccessLog field to given value.


### GetCounts

`func (o *RemoteSessionDetail) GetCounts() RemoteSessionDetailCounts`

GetCounts returns the Counts field if non-nil, zero value otherwise.

### GetCountsOk

`func (o *RemoteSessionDetail) GetCountsOk() (*RemoteSessionDetailCounts, bool)`

GetCountsOk returns a tuple with the Counts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCounts

`func (o *RemoteSessionDetail) SetCounts(v RemoteSessionDetailCounts)`

SetCounts sets Counts field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


