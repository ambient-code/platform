# InboxMessage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**AgentId** | **string** | Recipient — the agent address | 
**FromAgentId** | Pointer to **string** | Sender Agent id — null if sent by a human | [optional] 
**FromName** | Pointer to **string** | Denormalized sender display name | [optional] 
**Body** | **string** |  | 
**Read** | Pointer to **bool** | false &#x3D; unread; drained at session start | [optional] [readonly] 

## Methods

### NewInboxMessage

`func NewInboxMessage(agentId string, body string, ) *InboxMessage`

NewInboxMessage instantiates a new InboxMessage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewInboxMessageWithDefaults

`func NewInboxMessageWithDefaults() *InboxMessage`

NewInboxMessageWithDefaults instantiates a new InboxMessage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *InboxMessage) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *InboxMessage) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *InboxMessage) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *InboxMessage) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *InboxMessage) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *InboxMessage) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *InboxMessage) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *InboxMessage) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *InboxMessage) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *InboxMessage) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *InboxMessage) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *InboxMessage) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *InboxMessage) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *InboxMessage) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *InboxMessage) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *InboxMessage) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *InboxMessage) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *InboxMessage) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *InboxMessage) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *InboxMessage) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetAgentId

`func (o *InboxMessage) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *InboxMessage) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *InboxMessage) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.


### GetFromAgentId

`func (o *InboxMessage) GetFromAgentId() string`

GetFromAgentId returns the FromAgentId field if non-nil, zero value otherwise.

### GetFromAgentIdOk

`func (o *InboxMessage) GetFromAgentIdOk() (*string, bool)`

GetFromAgentIdOk returns a tuple with the FromAgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFromAgentId

`func (o *InboxMessage) SetFromAgentId(v string)`

SetFromAgentId sets FromAgentId field to given value.

### HasFromAgentId

`func (o *InboxMessage) HasFromAgentId() bool`

HasFromAgentId returns a boolean if a field has been set.

### GetFromName

`func (o *InboxMessage) GetFromName() string`

GetFromName returns the FromName field if non-nil, zero value otherwise.

### GetFromNameOk

`func (o *InboxMessage) GetFromNameOk() (*string, bool)`

GetFromNameOk returns a tuple with the FromName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFromName

`func (o *InboxMessage) SetFromName(v string)`

SetFromName sets FromName field to given value.

### HasFromName

`func (o *InboxMessage) HasFromName() bool`

HasFromName returns a boolean if a field has been set.

### GetBody

`func (o *InboxMessage) GetBody() string`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *InboxMessage) GetBodyOk() (*string, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *InboxMessage) SetBody(v string)`

SetBody sets Body field to given value.


### GetRead

`func (o *InboxMessage) GetRead() bool`

GetRead returns the Read field if non-nil, zero value otherwise.

### GetReadOk

`func (o *InboxMessage) GetReadOk() (*bool, bool)`

GetReadOk returns a tuple with the Read field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRead

`func (o *InboxMessage) SetRead(v bool)`

SetRead sets Read field to given value.

### HasRead

`func (o *InboxMessage) HasRead() bool`

HasRead returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


