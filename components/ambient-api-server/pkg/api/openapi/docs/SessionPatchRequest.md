# SessionPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**RepoUrl** | Pointer to **string** |  | [optional] 
**Prompt** | Pointer to **string** |  | [optional] 
**AssignedUserId** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**Repos** | Pointer to **string** |  | [optional] 
**Timeout** | Pointer to **int32** |  | [optional] 
**LlmModel** | Pointer to **string** |  | [optional] 
**LlmTemperature** | Pointer to **float64** |  | [optional] 
**LlmMaxTokens** | Pointer to **int32** |  | [optional] 
**ParentSessionId** | Pointer to **string** |  | [optional] 
**BotAccountName** | Pointer to **string** |  | [optional] 
**ResourceOverrides** | Pointer to **string** |  | [optional] 
**EnvironmentVariables** | Pointer to **string** |  | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 

## Methods

### NewSessionPatchRequest

`func NewSessionPatchRequest() *SessionPatchRequest`

NewSessionPatchRequest instantiates a new SessionPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionPatchRequestWithDefaults

`func NewSessionPatchRequestWithDefaults() *SessionPatchRequest`

NewSessionPatchRequestWithDefaults instantiates a new SessionPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *SessionPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *SessionPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *SessionPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *SessionPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRepoUrl

`func (o *SessionPatchRequest) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *SessionPatchRequest) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *SessionPatchRequest) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *SessionPatchRequest) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetPrompt

`func (o *SessionPatchRequest) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *SessionPatchRequest) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *SessionPatchRequest) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *SessionPatchRequest) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetAssignedUserId

`func (o *SessionPatchRequest) GetAssignedUserId() string`

GetAssignedUserId returns the AssignedUserId field if non-nil, zero value otherwise.

### GetAssignedUserIdOk

`func (o *SessionPatchRequest) GetAssignedUserIdOk() (*string, bool)`

GetAssignedUserIdOk returns a tuple with the AssignedUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAssignedUserId

`func (o *SessionPatchRequest) SetAssignedUserId(v string)`

SetAssignedUserId sets AssignedUserId field to given value.

### HasAssignedUserId

`func (o *SessionPatchRequest) HasAssignedUserId() bool`

HasAssignedUserId returns a boolean if a field has been set.

### GetWorkflowId

`func (o *SessionPatchRequest) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *SessionPatchRequest) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *SessionPatchRequest) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *SessionPatchRequest) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetRepos

`func (o *SessionPatchRequest) GetRepos() string`

GetRepos returns the Repos field if non-nil, zero value otherwise.

### GetReposOk

`func (o *SessionPatchRequest) GetReposOk() (*string, bool)`

GetReposOk returns a tuple with the Repos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepos

`func (o *SessionPatchRequest) SetRepos(v string)`

SetRepos sets Repos field to given value.

### HasRepos

`func (o *SessionPatchRequest) HasRepos() bool`

HasRepos returns a boolean if a field has been set.

### GetTimeout

`func (o *SessionPatchRequest) GetTimeout() int32`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *SessionPatchRequest) GetTimeoutOk() (*int32, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *SessionPatchRequest) SetTimeout(v int32)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *SessionPatchRequest) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetLlmModel

`func (o *SessionPatchRequest) GetLlmModel() string`

GetLlmModel returns the LlmModel field if non-nil, zero value otherwise.

### GetLlmModelOk

`func (o *SessionPatchRequest) GetLlmModelOk() (*string, bool)`

GetLlmModelOk returns a tuple with the LlmModel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmModel

`func (o *SessionPatchRequest) SetLlmModel(v string)`

SetLlmModel sets LlmModel field to given value.

### HasLlmModel

`func (o *SessionPatchRequest) HasLlmModel() bool`

HasLlmModel returns a boolean if a field has been set.

### GetLlmTemperature

`func (o *SessionPatchRequest) GetLlmTemperature() float64`

GetLlmTemperature returns the LlmTemperature field if non-nil, zero value otherwise.

### GetLlmTemperatureOk

`func (o *SessionPatchRequest) GetLlmTemperatureOk() (*float64, bool)`

GetLlmTemperatureOk returns a tuple with the LlmTemperature field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmTemperature

`func (o *SessionPatchRequest) SetLlmTemperature(v float64)`

SetLlmTemperature sets LlmTemperature field to given value.

### HasLlmTemperature

`func (o *SessionPatchRequest) HasLlmTemperature() bool`

HasLlmTemperature returns a boolean if a field has been set.

### GetLlmMaxTokens

`func (o *SessionPatchRequest) GetLlmMaxTokens() int32`

GetLlmMaxTokens returns the LlmMaxTokens field if non-nil, zero value otherwise.

### GetLlmMaxTokensOk

`func (o *SessionPatchRequest) GetLlmMaxTokensOk() (*int32, bool)`

GetLlmMaxTokensOk returns a tuple with the LlmMaxTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmMaxTokens

`func (o *SessionPatchRequest) SetLlmMaxTokens(v int32)`

SetLlmMaxTokens sets LlmMaxTokens field to given value.

### HasLlmMaxTokens

`func (o *SessionPatchRequest) HasLlmMaxTokens() bool`

HasLlmMaxTokens returns a boolean if a field has been set.

### GetParentSessionId

`func (o *SessionPatchRequest) GetParentSessionId() string`

GetParentSessionId returns the ParentSessionId field if non-nil, zero value otherwise.

### GetParentSessionIdOk

`func (o *SessionPatchRequest) GetParentSessionIdOk() (*string, bool)`

GetParentSessionIdOk returns a tuple with the ParentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParentSessionId

`func (o *SessionPatchRequest) SetParentSessionId(v string)`

SetParentSessionId sets ParentSessionId field to given value.

### HasParentSessionId

`func (o *SessionPatchRequest) HasParentSessionId() bool`

HasParentSessionId returns a boolean if a field has been set.

### GetBotAccountName

`func (o *SessionPatchRequest) GetBotAccountName() string`

GetBotAccountName returns the BotAccountName field if non-nil, zero value otherwise.

### GetBotAccountNameOk

`func (o *SessionPatchRequest) GetBotAccountNameOk() (*string, bool)`

GetBotAccountNameOk returns a tuple with the BotAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBotAccountName

`func (o *SessionPatchRequest) SetBotAccountName(v string)`

SetBotAccountName sets BotAccountName field to given value.

### HasBotAccountName

`func (o *SessionPatchRequest) HasBotAccountName() bool`

HasBotAccountName returns a boolean if a field has been set.

### GetResourceOverrides

`func (o *SessionPatchRequest) GetResourceOverrides() string`

GetResourceOverrides returns the ResourceOverrides field if non-nil, zero value otherwise.

### GetResourceOverridesOk

`func (o *SessionPatchRequest) GetResourceOverridesOk() (*string, bool)`

GetResourceOverridesOk returns a tuple with the ResourceOverrides field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceOverrides

`func (o *SessionPatchRequest) SetResourceOverrides(v string)`

SetResourceOverrides sets ResourceOverrides field to given value.

### HasResourceOverrides

`func (o *SessionPatchRequest) HasResourceOverrides() bool`

HasResourceOverrides returns a boolean if a field has been set.

### GetEnvironmentVariables

`func (o *SessionPatchRequest) GetEnvironmentVariables() string`

GetEnvironmentVariables returns the EnvironmentVariables field if non-nil, zero value otherwise.

### GetEnvironmentVariablesOk

`func (o *SessionPatchRequest) GetEnvironmentVariablesOk() (*string, bool)`

GetEnvironmentVariablesOk returns a tuple with the EnvironmentVariables field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvironmentVariables

`func (o *SessionPatchRequest) SetEnvironmentVariables(v string)`

SetEnvironmentVariables sets EnvironmentVariables field to given value.

### HasEnvironmentVariables

`func (o *SessionPatchRequest) HasEnvironmentVariables() bool`

HasEnvironmentVariables returns a boolean if a field has been set.

### GetLabels

`func (o *SessionPatchRequest) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *SessionPatchRequest) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *SessionPatchRequest) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *SessionPatchRequest) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *SessionPatchRequest) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *SessionPatchRequest) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *SessionPatchRequest) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *SessionPatchRequest) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


