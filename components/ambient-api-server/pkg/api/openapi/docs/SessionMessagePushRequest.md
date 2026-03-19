# SessionMessagePushRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EventType** | Pointer to **string** | Event type tag. Defaults to &#x60;user&#x60; if omitted. | [optional] [default to "user"]
**Payload** | Pointer to **string** | Message body | [optional] 

## Methods

### NewSessionMessagePushRequest

`func NewSessionMessagePushRequest() *SessionMessagePushRequest`

NewSessionMessagePushRequest instantiates a new SessionMessagePushRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionMessagePushRequestWithDefaults

`func NewSessionMessagePushRequestWithDefaults() *SessionMessagePushRequest`

NewSessionMessagePushRequestWithDefaults instantiates a new SessionMessagePushRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEventType

`func (o *SessionMessagePushRequest) GetEventType() string`

GetEventType returns the EventType field if non-nil, zero value otherwise.

### GetEventTypeOk

`func (o *SessionMessagePushRequest) GetEventTypeOk() (*string, bool)`

GetEventTypeOk returns a tuple with the EventType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventType

`func (o *SessionMessagePushRequest) SetEventType(v string)`

SetEventType sets EventType field to given value.

### HasEventType

`func (o *SessionMessagePushRequest) HasEventType() bool`

HasEventType returns a boolean if a field has been set.

### GetPayload

`func (o *SessionMessagePushRequest) GetPayload() string`

GetPayload returns the Payload field if non-nil, zero value otherwise.

### GetPayloadOk

`func (o *SessionMessagePushRequest) GetPayloadOk() (*string, bool)`

GetPayloadOk returns a tuple with the Payload field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayload

`func (o *SessionMessagePushRequest) SetPayload(v string)`

SetPayload sets Payload field to given value.

### HasPayload

`func (o *SessionMessagePushRequest) HasPayload() bool`

HasPayload returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


