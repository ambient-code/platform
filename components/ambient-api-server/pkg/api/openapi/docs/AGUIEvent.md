# AGUIEvent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**SessionId** | **string** |  | 
**Seq** | Pointer to **int64** |  | [optional] 
**EventType** | **string** |  | 
**Payload** | Pointer to **string** |  | [optional] 

## Methods

### NewAGUIEvent

`func NewAGUIEvent(sessionId string, eventType string, ) *AGUIEvent`

NewAGUIEvent instantiates a new AGUIEvent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAGUIEventWithDefaults

`func NewAGUIEventWithDefaults() *AGUIEvent`

NewAGUIEventWithDefaults instantiates a new AGUIEvent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *AGUIEvent) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AGUIEvent) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AGUIEvent) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *AGUIEvent) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *AGUIEvent) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *AGUIEvent) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *AGUIEvent) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *AGUIEvent) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *AGUIEvent) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *AGUIEvent) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *AGUIEvent) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *AGUIEvent) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AGUIEvent) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AGUIEvent) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AGUIEvent) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *AGUIEvent) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *AGUIEvent) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *AGUIEvent) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *AGUIEvent) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *AGUIEvent) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetSessionId

`func (o *AGUIEvent) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *AGUIEvent) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *AGUIEvent) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetSeq

`func (o *AGUIEvent) GetSeq() int64`

GetSeq returns the Seq field if non-nil, zero value otherwise.

### GetSeqOk

`func (o *AGUIEvent) GetSeqOk() (*int64, bool)`

GetSeqOk returns a tuple with the Seq field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeq

`func (o *AGUIEvent) SetSeq(v int64)`

SetSeq sets Seq field to given value.

### HasSeq

`func (o *AGUIEvent) HasSeq() bool`

HasSeq returns a boolean if a field has been set.

### GetEventType

`func (o *AGUIEvent) GetEventType() string`

GetEventType returns the EventType field if non-nil, zero value otherwise.

### GetEventTypeOk

`func (o *AGUIEvent) GetEventTypeOk() (*string, bool)`

GetEventTypeOk returns a tuple with the EventType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEventType

`func (o *AGUIEvent) SetEventType(v string)`

SetEventType sets EventType field to given value.


### GetPayload

`func (o *AGUIEvent) GetPayload() string`

GetPayload returns the Payload field if non-nil, zero value otherwise.

### GetPayloadOk

`func (o *AGUIEvent) GetPayloadOk() (*string, bool)`

GetPayloadOk returns a tuple with the Payload field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayload

`func (o *AGUIEvent) SetPayload(v string)`

SetPayload sets Payload field to given value.

### HasPayload

`func (o *AGUIEvent) HasPayload() bool`

HasPayload returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


