# RoleBindingPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RoleId** | Pointer to **string** |  | [optional] 
**Scope** | Pointer to **string** |  | [optional] 
**UserId** | Pointer to **NullableString** |  | [optional] 
**ProjectId** | Pointer to **NullableString** |  | [optional] 
**AgentId** | Pointer to **NullableString** |  | [optional] 
**SessionId** | Pointer to **NullableString** |  | [optional] 
**CredentialId** | Pointer to **NullableString** |  | [optional] 

## Methods

### NewRoleBindingPatchRequest

`func NewRoleBindingPatchRequest() *RoleBindingPatchRequest`

NewRoleBindingPatchRequest instantiates a new RoleBindingPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRoleBindingPatchRequestWithDefaults

`func NewRoleBindingPatchRequestWithDefaults() *RoleBindingPatchRequest`

NewRoleBindingPatchRequestWithDefaults instantiates a new RoleBindingPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRoleId

`func (o *RoleBindingPatchRequest) GetRoleId() string`

GetRoleId returns the RoleId field if non-nil, zero value otherwise.

### GetRoleIdOk

`func (o *RoleBindingPatchRequest) GetRoleIdOk() (*string, bool)`

GetRoleIdOk returns a tuple with the RoleId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRoleId

`func (o *RoleBindingPatchRequest) SetRoleId(v string)`

SetRoleId sets RoleId field to given value.

### HasRoleId

`func (o *RoleBindingPatchRequest) HasRoleId() bool`

HasRoleId returns a boolean if a field has been set.

### GetScope

`func (o *RoleBindingPatchRequest) GetScope() string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *RoleBindingPatchRequest) GetScopeOk() (*string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *RoleBindingPatchRequest) SetScope(v string)`

SetScope sets Scope field to given value.

### HasScope

`func (o *RoleBindingPatchRequest) HasScope() bool`

HasScope returns a boolean if a field has been set.

### GetUserId

`func (o *RoleBindingPatchRequest) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *RoleBindingPatchRequest) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *RoleBindingPatchRequest) SetUserId(v string)`

SetUserId sets UserId field to given value.

### HasUserId

`func (o *RoleBindingPatchRequest) HasUserId() bool`

HasUserId returns a boolean if a field has been set.

### SetUserIdNil

`func (o *RoleBindingPatchRequest) SetUserIdNil(b bool)`

 SetUserIdNil sets the value for UserId to be an explicit nil

### UnsetUserId
`func (o *RoleBindingPatchRequest) UnsetUserId()`

UnsetUserId ensures that no value is present for UserId, not even an explicit nil
### GetProjectId

`func (o *RoleBindingPatchRequest) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *RoleBindingPatchRequest) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *RoleBindingPatchRequest) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *RoleBindingPatchRequest) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.

### SetProjectIdNil

`func (o *RoleBindingPatchRequest) SetProjectIdNil(b bool)`

 SetProjectIdNil sets the value for ProjectId to be an explicit nil

### UnsetProjectId
`func (o *RoleBindingPatchRequest) UnsetProjectId()`

UnsetProjectId ensures that no value is present for ProjectId, not even an explicit nil
### GetAgentId

`func (o *RoleBindingPatchRequest) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *RoleBindingPatchRequest) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *RoleBindingPatchRequest) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.

### HasAgentId

`func (o *RoleBindingPatchRequest) HasAgentId() bool`

HasAgentId returns a boolean if a field has been set.

### SetAgentIdNil

`func (o *RoleBindingPatchRequest) SetAgentIdNil(b bool)`

 SetAgentIdNil sets the value for AgentId to be an explicit nil

### UnsetAgentId
`func (o *RoleBindingPatchRequest) UnsetAgentId()`

UnsetAgentId ensures that no value is present for AgentId, not even an explicit nil
### GetSessionId

`func (o *RoleBindingPatchRequest) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *RoleBindingPatchRequest) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *RoleBindingPatchRequest) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.

### HasSessionId

`func (o *RoleBindingPatchRequest) HasSessionId() bool`

HasSessionId returns a boolean if a field has been set.

### SetSessionIdNil

`func (o *RoleBindingPatchRequest) SetSessionIdNil(b bool)`

 SetSessionIdNil sets the value for SessionId to be an explicit nil

### UnsetSessionId
`func (o *RoleBindingPatchRequest) UnsetSessionId()`

UnsetSessionId ensures that no value is present for SessionId, not even an explicit nil
### GetCredentialId

`func (o *RoleBindingPatchRequest) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *RoleBindingPatchRequest) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *RoleBindingPatchRequest) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.

### HasCredentialId

`func (o *RoleBindingPatchRequest) HasCredentialId() bool`

HasCredentialId returns a boolean if a field has been set.

### SetCredentialIdNil

`func (o *RoleBindingPatchRequest) SetCredentialIdNil(b bool)`

 SetCredentialIdNil sets the value for CredentialId to be an explicit nil

### UnsetCredentialId
`func (o *RoleBindingPatchRequest) UnsetCredentialId()`

UnsetCredentialId ensures that no value is present for CredentialId, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


