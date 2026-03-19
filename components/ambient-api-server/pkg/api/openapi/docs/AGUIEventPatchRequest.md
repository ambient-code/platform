# AGUIEventPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SessionId** | Pointer to **string** |  | [optional] 
**Seq** | Pointer to **int64** |  | [optional] 
**EventType** | Pointer to **string** |  | [optional] 
**Payload** | Pointer to **string** |  | [optional] 

## Methods

### NewAGUIEventPatchRequest

`func NewAGUIEventPatchRequest() *AGUIEventPatchRequest`

NewAGUIEventPatchRequest instantiates a new AGUIEventPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAGUIEventPatchRequestWithDefaults

`func NewAGUIEventPatchRequestWithDefaults() *AGUIEventPatchRequest`

NewAGUIEventPatchRequestWithDefaults instantiates a new AGUIEventPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSessionId

`func (o *AGUIEventPatchRequest) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *AGUIEventPatchRequest) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *AGUIEventPatchRequest) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.

### HasSessionId

`func (o *AGUIEventPatchRequest) HasSessionId() bool`

HasSessionId returns a boolean if a field has been set.

### GetSeq

`func (o *AGUIEventPatchRequest) GetSeq() int64`

GetSeq returns the Seq field if non-nil, zero value otherwise.

### GetSeqOk

`func (o *AGUIEventPatchRequest) GetSeqOk() (*int64, bool)`

GetSeqOk returns a tuple with the Seq field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeq

`func (o *AGUIEventPatchRequest) SetSeq(v int64)`

SetSeq sets Seq field to given value.

### HasSeq

`func (o *AGUIEventPatchRequest) HasSeq() bool`

HasSeq returns a boolean if a field has been set.

### GetEventType

`func (o *AGUIEventPatchRequest) GetEventType() string`

GetEventType returns the EventType field if non-nil, zero value otherwise.

### GetEventTypeOk

`func (o *AGUIEventPatchRequest) GetEventTypeOk() (*string, bool)`

GetEventTypeOk returns a tuple with the EventType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventType

`func (o *AGUIEventPatchRequest) SetEventType(v string)`

SetEventType sets EventType field to given value.

### HasEventType

`func (o *AGUIEventPatchRequest) HasEventType() bool`

HasEventType returns a boolean if a field has been set.

### GetPayload

`func (o *AGUIEventPatchRequest) GetPayload() string`

GetPayload returns the Payload field if non-nil, zero value otherwise.

### GetPayloadOk

`func (o *AGUIEventPatchRequest) GetPayloadOk() (*string, bool)`

GetPayloadOk returns a tuple with the Payload field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayload

`func (o *AGUIEventPatchRequest) SetPayload(v string)`

SetPayload sets Payload field to given value.

### HasPayload

`func (o *AGUIEventPatchRequest) HasPayload() bool`

HasPayload returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


