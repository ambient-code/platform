# ProjectAgent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**ProjectId** | **string** |  | 
**AgentId** | **string** |  | 
**AgentVersion** | Pointer to **int32** | Pinned to a specific Agent version | [optional] 
**CurrentSessionId** | Pointer to **string** | Denormalized for fast reads — the active session, if any | [optional] [readonly] 

## Methods

### NewProjectAgent

`func NewProjectAgent(projectId string, agentId string, ) *ProjectAgent`

NewProjectAgent instantiates a new ProjectAgent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectAgentWithDefaults

`func NewProjectAgentWithDefaults() *ProjectAgent`

NewProjectAgentWithDefaults instantiates a new ProjectAgent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *ProjectAgent) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *ProjectAgent) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *ProjectAgent) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *ProjectAgent) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *ProjectAgent) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *ProjectAgent) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *ProjectAgent) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *ProjectAgent) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *ProjectAgent) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *ProjectAgent) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *ProjectAgent) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *ProjectAgent) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *ProjectAgent) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *ProjectAgent) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *ProjectAgent) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *ProjectAgent) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *ProjectAgent) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *ProjectAgent) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *ProjectAgent) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *ProjectAgent) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetProjectId

`func (o *ProjectAgent) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *ProjectAgent) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *ProjectAgent) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetAgentId

`func (o *ProjectAgent) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *ProjectAgent) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *ProjectAgent) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.


### GetAgentVersion

`func (o *ProjectAgent) GetAgentVersion() int32`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *ProjectAgent) GetAgentVersionOk() (*int32, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *ProjectAgent) SetAgentVersion(v int32)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *ProjectAgent) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.

### GetCurrentSessionId

`func (o *ProjectAgent) GetCurrentSessionId() string`

GetCurrentSessionId returns the CurrentSessionId field if non-nil, zero value otherwise.

### GetCurrentSessionIdOk

`func (o *ProjectAgent) GetCurrentSessionIdOk() (*string, bool)`

GetCurrentSessionIdOk returns a tuple with the CurrentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentSessionId

`func (o *ProjectAgent) SetCurrentSessionId(v string)`

SetCurrentSessionId sets CurrentSessionId field to given value.

### HasCurrentSessionId

`func (o *ProjectAgent) HasCurrentSessionId() bool`

HasCurrentSessionId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


