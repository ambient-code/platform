# Agent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**ProjectId** | **string** |  | 
**ParentAgentId** | Pointer to **string** |  | [optional] 
**OwnerUserId** | **string** |  | 
**Name** | **string** |  | 
**DisplayName** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Prompt** | Pointer to **string** |  | [optional] 
**RepoUrl** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**LlmModel** | Pointer to **string** |  | [optional] 
**LlmTemperature** | Pointer to **float64** |  | [optional] 
**LlmMaxTokens** | Pointer to **int32** |  | [optional] 
**BotAccountName** | Pointer to **string** |  | [optional] 
**ResourceOverrides** | Pointer to **string** |  | [optional] 
**EnvironmentVariables** | Pointer to **string** |  | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 
**CurrentSessionId** | Pointer to **string** |  | [optional] 

## Methods

### NewAgent

`func NewAgent(projectId string, ownerUserId string, name string, ) *Agent`

NewAgent instantiates a new Agent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentWithDefaults

`func NewAgentWithDefaults() *Agent`

NewAgentWithDefaults instantiates a new Agent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Agent) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Agent) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Agent) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Agent) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Agent) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Agent) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Agent) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Agent) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Agent) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Agent) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Agent) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Agent) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Agent) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Agent) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Agent) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Agent) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Agent) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Agent) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Agent) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Agent) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetProjectId

`func (o *Agent) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *Agent) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *Agent) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetParentAgentId

`func (o *Agent) GetParentAgentId() string`

GetParentAgentId returns the ParentAgentId field if non-nil, zero value otherwise.

### GetParentAgentIdOk

`func (o *Agent) GetParentAgentIdOk() (*string, bool)`

GetParentAgentIdOk returns a tuple with the ParentAgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParentAgentId

`func (o *Agent) SetParentAgentId(v string)`

SetParentAgentId sets ParentAgentId field to given value.

### HasParentAgentId

`func (o *Agent) HasParentAgentId() bool`

HasParentAgentId returns a boolean if a field has been set.

### GetOwnerUserId

`func (o *Agent) GetOwnerUserId() string`

GetOwnerUserId returns the OwnerUserId field if non-nil, zero value otherwise.

### GetOwnerUserIdOk

`func (o *Agent) GetOwnerUserIdOk() (*string, bool)`

GetOwnerUserIdOk returns a tuple with the OwnerUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwnerUserId

`func (o *Agent) SetOwnerUserId(v string)`

SetOwnerUserId sets OwnerUserId field to given value.


### GetName

`func (o *Agent) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Agent) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Agent) SetName(v string)`

SetName sets Name field to given value.


### GetDisplayName

`func (o *Agent) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *Agent) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *Agent) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *Agent) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetDescription

`func (o *Agent) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *Agent) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *Agent) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *Agent) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetPrompt

`func (o *Agent) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *Agent) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *Agent) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *Agent) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetRepoUrl

`func (o *Agent) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *Agent) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *Agent) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *Agent) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetWorkflowId

`func (o *Agent) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *Agent) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *Agent) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *Agent) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetLlmModel

`func (o *Agent) GetLlmModel() string`

GetLlmModel returns the LlmModel field if non-nil, zero value otherwise.

### GetLlmModelOk

`func (o *Agent) GetLlmModelOk() (*string, bool)`

GetLlmModelOk returns a tuple with the LlmModel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmModel

`func (o *Agent) SetLlmModel(v string)`

SetLlmModel sets LlmModel field to given value.

### HasLlmModel

`func (o *Agent) HasLlmModel() bool`

HasLlmModel returns a boolean if a field has been set.

### GetLlmTemperature

`func (o *Agent) GetLlmTemperature() float64`

GetLlmTemperature returns the LlmTemperature field if non-nil, zero value otherwise.

### GetLlmTemperatureOk

`func (o *Agent) GetLlmTemperatureOk() (*float64, bool)`

GetLlmTemperatureOk returns a tuple with the LlmTemperature field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmTemperature

`func (o *Agent) SetLlmTemperature(v float64)`

SetLlmTemperature sets LlmTemperature field to given value.

### HasLlmTemperature

`func (o *Agent) HasLlmTemperature() bool`

HasLlmTemperature returns a boolean if a field has been set.

### GetLlmMaxTokens

`func (o *Agent) GetLlmMaxTokens() int32`

GetLlmMaxTokens returns the LlmMaxTokens field if non-nil, zero value otherwise.

### GetLlmMaxTokensOk

`func (o *Agent) GetLlmMaxTokensOk() (*int32, bool)`

GetLlmMaxTokensOk returns a tuple with the LlmMaxTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmMaxTokens

`func (o *Agent) SetLlmMaxTokens(v int32)`

SetLlmMaxTokens sets LlmMaxTokens field to given value.

### HasLlmMaxTokens

`func (o *Agent) HasLlmMaxTokens() bool`

HasLlmMaxTokens returns a boolean if a field has been set.

### GetBotAccountName

`func (o *Agent) GetBotAccountName() string`

GetBotAccountName returns the BotAccountName field if non-nil, zero value otherwise.

### GetBotAccountNameOk

`func (o *Agent) GetBotAccountNameOk() (*string, bool)`

GetBotAccountNameOk returns a tuple with the BotAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBotAccountName

`func (o *Agent) SetBotAccountName(v string)`

SetBotAccountName sets BotAccountName field to given value.

### HasBotAccountName

`func (o *Agent) HasBotAccountName() bool`

HasBotAccountName returns a boolean if a field has been set.

### GetResourceOverrides

`func (o *Agent) GetResourceOverrides() string`

GetResourceOverrides returns the ResourceOverrides field if non-nil, zero value otherwise.

### GetResourceOverridesOk

`func (o *Agent) GetResourceOverridesOk() (*string, bool)`

GetResourceOverridesOk returns a tuple with the ResourceOverrides field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceOverrides

`func (o *Agent) SetResourceOverrides(v string)`

SetResourceOverrides sets ResourceOverrides field to given value.

### HasResourceOverrides

`func (o *Agent) HasResourceOverrides() bool`

HasResourceOverrides returns a boolean if a field has been set.

### GetEnvironmentVariables

`func (o *Agent) GetEnvironmentVariables() string`

GetEnvironmentVariables returns the EnvironmentVariables field if non-nil, zero value otherwise.

### GetEnvironmentVariablesOk

`func (o *Agent) GetEnvironmentVariablesOk() (*string, bool)`

GetEnvironmentVariablesOk returns a tuple with the EnvironmentVariables field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvironmentVariables

`func (o *Agent) SetEnvironmentVariables(v string)`

SetEnvironmentVariables sets EnvironmentVariables field to given value.

### HasEnvironmentVariables

`func (o *Agent) HasEnvironmentVariables() bool`

HasEnvironmentVariables returns a boolean if a field has been set.

### GetLabels

`func (o *Agent) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *Agent) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *Agent) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *Agent) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *Agent) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *Agent) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *Agent) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *Agent) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetCurrentSessionId

`func (o *Agent) GetCurrentSessionId() string`

GetCurrentSessionId returns the CurrentSessionId field if non-nil, zero value otherwise.

### GetCurrentSessionIdOk

`func (o *Agent) GetCurrentSessionIdOk() (*string, bool)`

GetCurrentSessionIdOk returns a tuple with the CurrentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentSessionId

`func (o *Agent) SetCurrentSessionId(v string)`

SetCurrentSessionId sets CurrentSessionId field to given value.

### HasCurrentSessionId

`func (o *Agent) HasCurrentSessionId() bool`

HasCurrentSessionId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


