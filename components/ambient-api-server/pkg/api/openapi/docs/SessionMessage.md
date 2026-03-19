# SessionMessage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**SessionId** | Pointer to **string** | ID of the parent session | [optional] [readonly] 
**Seq** | Pointer to **int64** | Monotonically increasing sequence number within the session | [optional] [readonly] 
**EventType** | Pointer to **string** | Event type tag. Common values: &#x60;user&#x60; (human turn), &#x60;assistant&#x60; (model reply), &#x60;tool_use&#x60;, &#x60;tool_result&#x60;, &#x60;system&#x60;, &#x60;error&#x60;. | [optional] [default to "user"]
**Payload** | Pointer to **string** | Message body (plain text or JSON-encoded event payload) | [optional] 

## Methods

### NewSessionMessage

`func NewSessionMessage() *SessionMessage`

NewSessionMessage instantiates a new SessionMessage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionMessageWithDefaults

`func NewSessionMessageWithDefaults() *SessionMessage`

NewSessionMessageWithDefaults instantiates a new SessionMessage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SessionMessage) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SessionMessage) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SessionMessage) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *SessionMessage) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *SessionMessage) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *SessionMessage) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *SessionMessage) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *SessionMessage) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *SessionMessage) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *SessionMessage) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *SessionMessage) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *SessionMessage) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *SessionMessage) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SessionMessage) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SessionMessage) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *SessionMessage) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *SessionMessage) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *SessionMessage) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *SessionMessage) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *SessionMessage) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetSessionId

`func (o *SessionMessage) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *SessionMessage) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *SessionMessage) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.

### HasSessionId

`func (o *SessionMessage) HasSessionId() bool`

HasSessionId returns a boolean if a field has been set.

### GetSeq

`func (o *SessionMessage) GetSeq() int64`

GetSeq returns the Seq field if non-nil, zero value otherwise.

### GetSeqOk

`func (o *SessionMessage) GetSeqOk() (*int64, bool)`

GetSeqOk returns a tuple with the Seq field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeq

`func (o *SessionMessage) SetSeq(v int64)`

SetSeq sets Seq field to given value.

### HasSeq

`func (o *SessionMessage) HasSeq() bool`

HasSeq returns a boolean if a field has been set.

### GetEventType

`func (o *SessionMessage) GetEventType() string`

GetEventType returns the EventType field if non-nil, zero value otherwise.

### GetEventTypeOk

`func (o *SessionMessage) GetEventTypeOk() (*string, bool)`

GetEventTypeOk returns a tuple with the EventType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventType

`func (o *SessionMessage) SetEventType(v string)`

SetEventType sets EventType field to given value.

### HasEventType

`func (o *SessionMessage) HasEventType() bool`

HasEventType returns a boolean if a field has been set.

### GetPayload

`func (o *SessionMessage) GetPayload() string`

GetPayload returns the Payload field if non-nil, zero value otherwise.

### GetPayloadOk

`func (o *SessionMessage) GetPayloadOk() (*string, bool)`

GetPayloadOk returns a tuple with the Payload field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayload

`func (o *SessionMessage) SetPayload(v string)`

SetPayload sets Payload field to given value.

### HasPayload

`func (o *SessionMessage) HasPayload() bool`

HasPayload returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


