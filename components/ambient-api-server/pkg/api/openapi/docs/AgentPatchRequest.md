# AgentPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ProjectId** | Pointer to **string** |  | [optional] 
**ParentAgentId** | Pointer to **string** |  | [optional] 
**OwnerUserId** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
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

### NewAgentPatchRequest

`func NewAgentPatchRequest() *AgentPatchRequest`

NewAgentPatchRequest instantiates a new AgentPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentPatchRequestWithDefaults

`func NewAgentPatchRequestWithDefaults() *AgentPatchRequest`

NewAgentPatchRequestWithDefaults instantiates a new AgentPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProjectId

`func (o *AgentPatchRequest) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *AgentPatchRequest) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *AgentPatchRequest) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *AgentPatchRequest) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.

### GetParentAgentId

`func (o *AgentPatchRequest) GetParentAgentId() string`

GetParentAgentId returns the ParentAgentId field if non-nil, zero value otherwise.

### GetParentAgentIdOk

`func (o *AgentPatchRequest) GetParentAgentIdOk() (*string, bool)`

GetParentAgentIdOk returns a tuple with the ParentAgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParentAgentId

`func (o *AgentPatchRequest) SetParentAgentId(v string)`

SetParentAgentId sets ParentAgentId field to given value.

### HasParentAgentId

`func (o *AgentPatchRequest) HasParentAgentId() bool`

HasParentAgentId returns a boolean if a field has been set.

### GetOwnerUserId

`func (o *AgentPatchRequest) GetOwnerUserId() string`

GetOwnerUserId returns the OwnerUserId field if non-nil, zero value otherwise.

### GetOwnerUserIdOk

`func (o *AgentPatchRequest) GetOwnerUserIdOk() (*string, bool)`

GetOwnerUserIdOk returns a tuple with the OwnerUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwnerUserId

`func (o *AgentPatchRequest) SetOwnerUserId(v string)`

SetOwnerUserId sets OwnerUserId field to given value.

### HasOwnerUserId

`func (o *AgentPatchRequest) HasOwnerUserId() bool`

HasOwnerUserId returns a boolean if a field has been set.

### GetName

`func (o *AgentPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AgentPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AgentPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *AgentPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDisplayName

`func (o *AgentPatchRequest) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *AgentPatchRequest) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *AgentPatchRequest) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *AgentPatchRequest) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetDescription

`func (o *AgentPatchRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *AgentPatchRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *AgentPatchRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *AgentPatchRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetPrompt

`func (o *AgentPatchRequest) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *AgentPatchRequest) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *AgentPatchRequest) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *AgentPatchRequest) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetRepoUrl

`func (o *AgentPatchRequest) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *AgentPatchRequest) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *AgentPatchRequest) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *AgentPatchRequest) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetWorkflowId

`func (o *AgentPatchRequest) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *AgentPatchRequest) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *AgentPatchRequest) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *AgentPatchRequest) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetLlmModel

`func (o *AgentPatchRequest) GetLlmModel() string`

GetLlmModel returns the LlmModel field if non-nil, zero value otherwise.

### GetLlmModelOk

`func (o *AgentPatchRequest) GetLlmModelOk() (*string, bool)`

GetLlmModelOk returns a tuple with the LlmModel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmModel

`func (o *AgentPatchRequest) SetLlmModel(v string)`

SetLlmModel sets LlmModel field to given value.

### HasLlmModel

`func (o *AgentPatchRequest) HasLlmModel() bool`

HasLlmModel returns a boolean if a field has been set.

### GetLlmTemperature

`func (o *AgentPatchRequest) GetLlmTemperature() float64`

GetLlmTemperature returns the LlmTemperature field if non-nil, zero value otherwise.

### GetLlmTemperatureOk

`func (o *AgentPatchRequest) GetLlmTemperatureOk() (*float64, bool)`

GetLlmTemperatureOk returns a tuple with the LlmTemperature field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmTemperature

`func (o *AgentPatchRequest) SetLlmTemperature(v float64)`

SetLlmTemperature sets LlmTemperature field to given value.

### HasLlmTemperature

`func (o *AgentPatchRequest) HasLlmTemperature() bool`

HasLlmTemperature returns a boolean if a field has been set.

### GetLlmMaxTokens

`func (o *AgentPatchRequest) GetLlmMaxTokens() int32`

GetLlmMaxTokens returns the LlmMaxTokens field if non-nil, zero value otherwise.

### GetLlmMaxTokensOk

`func (o *AgentPatchRequest) GetLlmMaxTokensOk() (*int32, bool)`

GetLlmMaxTokensOk returns a tuple with the LlmMaxTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmMaxTokens

`func (o *AgentPatchRequest) SetLlmMaxTokens(v int32)`

SetLlmMaxTokens sets LlmMaxTokens field to given value.

### HasLlmMaxTokens

`func (o *AgentPatchRequest) HasLlmMaxTokens() bool`

HasLlmMaxTokens returns a boolean if a field has been set.

### GetBotAccountName

`func (o *AgentPatchRequest) GetBotAccountName() string`

GetBotAccountName returns the BotAccountName field if non-nil, zero value otherwise.

### GetBotAccountNameOk

`func (o *AgentPatchRequest) GetBotAccountNameOk() (*string, bool)`

GetBotAccountNameOk returns a tuple with the BotAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBotAccountName

`func (o *AgentPatchRequest) SetBotAccountName(v string)`

SetBotAccountName sets BotAccountName field to given value.

### HasBotAccountName

`func (o *AgentPatchRequest) HasBotAccountName() bool`

HasBotAccountName returns a boolean if a field has been set.

### GetResourceOverrides

`func (o *AgentPatchRequest) GetResourceOverrides() string`

GetResourceOverrides returns the ResourceOverrides field if non-nil, zero value otherwise.

### GetResourceOverridesOk

`func (o *AgentPatchRequest) GetResourceOverridesOk() (*string, bool)`

GetResourceOverridesOk returns a tuple with the ResourceOverrides field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceOverrides

`func (o *AgentPatchRequest) SetResourceOverrides(v string)`

SetResourceOverrides sets ResourceOverrides field to given value.

### HasResourceOverrides

`func (o *AgentPatchRequest) HasResourceOverrides() bool`

HasResourceOverrides returns a boolean if a field has been set.

### GetEnvironmentVariables

`func (o *AgentPatchRequest) GetEnvironmentVariables() string`

GetEnvironmentVariables returns the EnvironmentVariables field if non-nil, zero value otherwise.

### GetEnvironmentVariablesOk

`func (o *AgentPatchRequest) GetEnvironmentVariablesOk() (*string, bool)`

GetEnvironmentVariablesOk returns a tuple with the EnvironmentVariables field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvironmentVariables

`func (o *AgentPatchRequest) SetEnvironmentVariables(v string)`

SetEnvironmentVariables sets EnvironmentVariables field to given value.

### HasEnvironmentVariables

`func (o *AgentPatchRequest) HasEnvironmentVariables() bool`

HasEnvironmentVariables returns a boolean if a field has been set.

### GetLabels

`func (o *AgentPatchRequest) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *AgentPatchRequest) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *AgentPatchRequest) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *AgentPatchRequest) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *AgentPatchRequest) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *AgentPatchRequest) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *AgentPatchRequest) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *AgentPatchRequest) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetCurrentSessionId

`func (o *AgentPatchRequest) GetCurrentSessionId() string`

GetCurrentSessionId returns the CurrentSessionId field if non-nil, zero value otherwise.

### GetCurrentSessionIdOk

`func (o *AgentPatchRequest) GetCurrentSessionIdOk() (*string, bool)`

GetCurrentSessionIdOk returns a tuple with the CurrentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentSessionId

`func (o *AgentPatchRequest) SetCurrentSessionId(v string)`

SetCurrentSessionId sets CurrentSessionId field to given value.

### HasCurrentSessionId

`func (o *AgentPatchRequest) HasCurrentSessionId() bool`

HasCurrentSessionId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


