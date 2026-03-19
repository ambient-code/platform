# AgentMessage

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**RecipientAgentId** | **string** |  | 
**SenderAgentId** | Pointer to **string** |  | [optional] 
**SenderUserId** | Pointer to **string** |  | [optional] 
**SenderName** | Pointer to **string** |  | [optional] 
**Body** | Pointer to **string** |  | [optional] 
**Read** | Pointer to **bool** |  | [optional] 

## Methods

### NewAgentMessage

`func NewAgentMessage(recipientAgentId string, ) *AgentMessage`

NewAgentMessage instantiates a new AgentMessage object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentMessageWithDefaults

`func NewAgentMessageWithDefaults() *AgentMessage`

NewAgentMessageWithDefaults instantiates a new AgentMessage object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *AgentMessage) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AgentMessage) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AgentMessage) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *AgentMessage) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *AgentMessage) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *AgentMessage) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *AgentMessage) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *AgentMessage) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *AgentMessage) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *AgentMessage) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *AgentMessage) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *AgentMessage) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AgentMessage) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AgentMessage) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AgentMessage) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *AgentMessage) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *AgentMessage) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *AgentMessage) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *AgentMessage) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *AgentMessage) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetRecipientAgentId

`func (o *AgentMessage) GetRecipientAgentId() string`

GetRecipientAgentId returns the RecipientAgentId field if non-nil, zero value otherwise.

### GetRecipientAgentIdOk

`func (o *AgentMessage) GetRecipientAgentIdOk() (*string, bool)`

GetRecipientAgentIdOk returns a tuple with the RecipientAgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRecipientAgentId

`func (o *AgentMessage) SetRecipientAgentId(v string)`

SetRecipientAgentId sets RecipientAgentId field to given value.


### GetSenderAgentId

`func (o *AgentMessage) GetSenderAgentId() string`

GetSenderAgentId returns the SenderAgentId field if non-nil, zero value otherwise.

### GetSenderAgentIdOk

`func (o *AgentMessage) GetSenderAgentIdOk() (*string, bool)`

GetSenderAgentIdOk returns a tuple with the SenderAgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSenderAgentId

`func (o *AgentMessage) SetSenderAgentId(v string)`

SetSenderAgentId sets SenderAgentId field to given value.

### HasSenderAgentId

`func (o *AgentMessage) HasSenderAgentId() bool`

HasSenderAgentId returns a boolean if a field has been set.

### GetSenderUserId

`func (o *AgentMessage) GetSenderUserId() string`

GetSenderUserId returns the SenderUserId field if non-nil, zero value otherwise.

### GetSenderUserIdOk

`func (o *AgentMessage) GetSenderUserIdOk() (*string, bool)`

GetSenderUserIdOk returns a tuple with the SenderUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSenderUserId

`func (o *AgentMessage) SetSenderUserId(v string)`

SetSenderUserId sets SenderUserId field to given value.

### HasSenderUserId

`func (o *AgentMessage) HasSenderUserId() bool`

HasSenderUserId returns a boolean if a field has been set.

### GetSenderName

`func (o *AgentMessage) GetSenderName() string`

GetSenderName returns the SenderName field if non-nil, zero value otherwise.

### GetSenderNameOk

`func (o *AgentMessage) GetSenderNameOk() (*string, bool)`

GetSenderNameOk returns a tuple with the SenderName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSenderName

`func (o *AgentMessage) SetSenderName(v string)`

SetSenderName sets SenderName field to given value.

### HasSenderName

`func (o *AgentMessage) HasSenderName() bool`

HasSenderName returns a boolean if a field has been set.

### GetBody

`func (o *AgentMessage) GetBody() string`

GetBody returns the Body field if non-nil, zero value otherwise.

### GetBodyOk

`func (o *AgentMessage) GetBodyOk() (*string, bool)`

GetBodyOk returns a tuple with the Body field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBody

`func (o *AgentMessage) SetBody(v string)`

SetBody sets Body field to given value.

### HasBody

`func (o *AgentMessage) HasBody() bool`

HasBody returns a boolean if a field has been set.

### GetRead

`func (o *AgentMessage) GetRead() bool`

GetRead returns the Read field if non-nil, zero value otherwise.

### GetReadOk

`func (o *AgentMessage) GetReadOk() (*bool, bool)`

GetReadOk returns a tuple with the Read field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRead

`func (o *AgentMessage) SetRead(v bool)`

SetRead sets Read field to given value.

### HasRead

`func (o *AgentMessage) HasRead() bool`

HasRead returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


