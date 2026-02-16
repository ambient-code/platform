# Session

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Name** | **string** |  | 
**RepoUrl** | Pointer to **string** |  | [optional] 
**Prompt** | Pointer to **string** |  | [optional] 
**CreatedByUserId** | Pointer to **string** | Set from authentication token. Cannot be set or modified via API. | [optional] [readonly] 
**AssignedUserId** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**Repos** | Pointer to **string** |  | [optional] 
**Interactive** | Pointer to **bool** |  | [optional] 
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
**ProjectId** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] [readonly] 
**StartTime** | Pointer to **time.Time** |  | [optional] [readonly] 
**CompletionTime** | Pointer to **time.Time** |  | [optional] [readonly] 
**SdkSessionId** | Pointer to **string** |  | [optional] [readonly] 
**SdkRestartCount** | Pointer to **int32** |  | [optional] [readonly] 
**Conditions** | Pointer to **string** |  | [optional] [readonly] 
**ReconciledRepos** | Pointer to **string** |  | [optional] [readonly] 
**ReconciledWorkflow** | Pointer to **string** |  | [optional] [readonly] 
**KubeCrName** | Pointer to **string** |  | [optional] [readonly] 
**KubeCrUid** | Pointer to **string** |  | [optional] [readonly] 
**KubeNamespace** | Pointer to **string** |  | [optional] [readonly] 

## Methods

### NewSession

`func NewSession(name string, ) *Session`

NewSession instantiates a new Session object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionWithDefaults

`func NewSessionWithDefaults() *Session`

NewSessionWithDefaults instantiates a new Session object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Session) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Session) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Session) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Session) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Session) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Session) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Session) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Session) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Session) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Session) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Session) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Session) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Session) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Session) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Session) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Session) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Session) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Session) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Session) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Session) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetName

`func (o *Session) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Session) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Session) SetName(v string)`

SetName sets Name field to given value.


### GetRepoUrl

`func (o *Session) GetRepoUrl() string`

GetRepoUrl returns the RepoUrl field if non-nil, zero value otherwise.

### GetRepoUrlOk

`func (o *Session) GetRepoUrlOk() (*string, bool)`

GetRepoUrlOk returns a tuple with the RepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoUrl

`func (o *Session) SetRepoUrl(v string)`

SetRepoUrl sets RepoUrl field to given value.

### HasRepoUrl

`func (o *Session) HasRepoUrl() bool`

HasRepoUrl returns a boolean if a field has been set.

### GetPrompt

`func (o *Session) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *Session) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *Session) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *Session) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetCreatedByUserId

`func (o *Session) GetCreatedByUserId() string`

GetCreatedByUserId returns the CreatedByUserId field if non-nil, zero value otherwise.

### GetCreatedByUserIdOk

`func (o *Session) GetCreatedByUserIdOk() (*string, bool)`

GetCreatedByUserIdOk returns a tuple with the CreatedByUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedByUserId

`func (o *Session) SetCreatedByUserId(v string)`

SetCreatedByUserId sets CreatedByUserId field to given value.

### HasCreatedByUserId

`func (o *Session) HasCreatedByUserId() bool`

HasCreatedByUserId returns a boolean if a field has been set.

### GetAssignedUserId

`func (o *Session) GetAssignedUserId() string`

GetAssignedUserId returns the AssignedUserId field if non-nil, zero value otherwise.

### GetAssignedUserIdOk

`func (o *Session) GetAssignedUserIdOk() (*string, bool)`

GetAssignedUserIdOk returns a tuple with the AssignedUserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAssignedUserId

`func (o *Session) SetAssignedUserId(v string)`

SetAssignedUserId sets AssignedUserId field to given value.

### HasAssignedUserId

`func (o *Session) HasAssignedUserId() bool`

HasAssignedUserId returns a boolean if a field has been set.

### GetWorkflowId

`func (o *Session) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *Session) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *Session) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *Session) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetRepos

`func (o *Session) GetRepos() string`

GetRepos returns the Repos field if non-nil, zero value otherwise.

### GetReposOk

`func (o *Session) GetReposOk() (*string, bool)`

GetReposOk returns a tuple with the Repos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepos

`func (o *Session) SetRepos(v string)`

SetRepos sets Repos field to given value.

### HasRepos

`func (o *Session) HasRepos() bool`

HasRepos returns a boolean if a field has been set.

### GetInteractive

`func (o *Session) GetInteractive() bool`

GetInteractive returns the Interactive field if non-nil, zero value otherwise.

### GetInteractiveOk

`func (o *Session) GetInteractiveOk() (*bool, bool)`

GetInteractiveOk returns a tuple with the Interactive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInteractive

`func (o *Session) SetInteractive(v bool)`

SetInteractive sets Interactive field to given value.

### HasInteractive

`func (o *Session) HasInteractive() bool`

HasInteractive returns a boolean if a field has been set.

### GetTimeout

`func (o *Session) GetTimeout() int32`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *Session) GetTimeoutOk() (*int32, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *Session) SetTimeout(v int32)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *Session) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetLlmModel

`func (o *Session) GetLlmModel() string`

GetLlmModel returns the LlmModel field if non-nil, zero value otherwise.

### GetLlmModelOk

`func (o *Session) GetLlmModelOk() (*string, bool)`

GetLlmModelOk returns a tuple with the LlmModel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmModel

`func (o *Session) SetLlmModel(v string)`

SetLlmModel sets LlmModel field to given value.

### HasLlmModel

`func (o *Session) HasLlmModel() bool`

HasLlmModel returns a boolean if a field has been set.

### GetLlmTemperature

`func (o *Session) GetLlmTemperature() float64`

GetLlmTemperature returns the LlmTemperature field if non-nil, zero value otherwise.

### GetLlmTemperatureOk

`func (o *Session) GetLlmTemperatureOk() (*float64, bool)`

GetLlmTemperatureOk returns a tuple with the LlmTemperature field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmTemperature

`func (o *Session) SetLlmTemperature(v float64)`

SetLlmTemperature sets LlmTemperature field to given value.

### HasLlmTemperature

`func (o *Session) HasLlmTemperature() bool`

HasLlmTemperature returns a boolean if a field has been set.

### GetLlmMaxTokens

`func (o *Session) GetLlmMaxTokens() int32`

GetLlmMaxTokens returns the LlmMaxTokens field if non-nil, zero value otherwise.

### GetLlmMaxTokensOk

`func (o *Session) GetLlmMaxTokensOk() (*int32, bool)`

GetLlmMaxTokensOk returns a tuple with the LlmMaxTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLlmMaxTokens

`func (o *Session) SetLlmMaxTokens(v int32)`

SetLlmMaxTokens sets LlmMaxTokens field to given value.

### HasLlmMaxTokens

`func (o *Session) HasLlmMaxTokens() bool`

HasLlmMaxTokens returns a boolean if a field has been set.

### GetParentSessionId

`func (o *Session) GetParentSessionId() string`

GetParentSessionId returns the ParentSessionId field if non-nil, zero value otherwise.

### GetParentSessionIdOk

`func (o *Session) GetParentSessionIdOk() (*string, bool)`

GetParentSessionIdOk returns a tuple with the ParentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParentSessionId

`func (o *Session) SetParentSessionId(v string)`

SetParentSessionId sets ParentSessionId field to given value.

### HasParentSessionId

`func (o *Session) HasParentSessionId() bool`

HasParentSessionId returns a boolean if a field has been set.

### GetBotAccountName

`func (o *Session) GetBotAccountName() string`

GetBotAccountName returns the BotAccountName field if non-nil, zero value otherwise.

### GetBotAccountNameOk

`func (o *Session) GetBotAccountNameOk() (*string, bool)`

GetBotAccountNameOk returns a tuple with the BotAccountName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBotAccountName

`func (o *Session) SetBotAccountName(v string)`

SetBotAccountName sets BotAccountName field to given value.

### HasBotAccountName

`func (o *Session) HasBotAccountName() bool`

HasBotAccountName returns a boolean if a field has been set.

### GetResourceOverrides

`func (o *Session) GetResourceOverrides() string`

GetResourceOverrides returns the ResourceOverrides field if non-nil, zero value otherwise.

### GetResourceOverridesOk

`func (o *Session) GetResourceOverridesOk() (*string, bool)`

GetResourceOverridesOk returns a tuple with the ResourceOverrides field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceOverrides

`func (o *Session) SetResourceOverrides(v string)`

SetResourceOverrides sets ResourceOverrides field to given value.

### HasResourceOverrides

`func (o *Session) HasResourceOverrides() bool`

HasResourceOverrides returns a boolean if a field has been set.

### GetEnvironmentVariables

`func (o *Session) GetEnvironmentVariables() string`

GetEnvironmentVariables returns the EnvironmentVariables field if non-nil, zero value otherwise.

### GetEnvironmentVariablesOk

`func (o *Session) GetEnvironmentVariablesOk() (*string, bool)`

GetEnvironmentVariablesOk returns a tuple with the EnvironmentVariables field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvironmentVariables

`func (o *Session) SetEnvironmentVariables(v string)`

SetEnvironmentVariables sets EnvironmentVariables field to given value.

### HasEnvironmentVariables

`func (o *Session) HasEnvironmentVariables() bool`

HasEnvironmentVariables returns a boolean if a field has been set.

### GetLabels

`func (o *Session) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *Session) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *Session) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *Session) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *Session) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *Session) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *Session) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *Session) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetProjectId

`func (o *Session) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *Session) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *Session) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *Session) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.

### GetPhase

`func (o *Session) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *Session) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *Session) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *Session) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStartTime

`func (o *Session) GetStartTime() time.Time`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *Session) GetStartTimeOk() (*time.Time, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *Session) SetStartTime(v time.Time)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *Session) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.

### GetCompletionTime

`func (o *Session) GetCompletionTime() time.Time`

GetCompletionTime returns the CompletionTime field if non-nil, zero value otherwise.

### GetCompletionTimeOk

`func (o *Session) GetCompletionTimeOk() (*time.Time, bool)`

GetCompletionTimeOk returns a tuple with the CompletionTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletionTime

`func (o *Session) SetCompletionTime(v time.Time)`

SetCompletionTime sets CompletionTime field to given value.

### HasCompletionTime

`func (o *Session) HasCompletionTime() bool`

HasCompletionTime returns a boolean if a field has been set.

### GetSdkSessionId

`func (o *Session) GetSdkSessionId() string`

GetSdkSessionId returns the SdkSessionId field if non-nil, zero value otherwise.

### GetSdkSessionIdOk

`func (o *Session) GetSdkSessionIdOk() (*string, bool)`

GetSdkSessionIdOk returns a tuple with the SdkSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSdkSessionId

`func (o *Session) SetSdkSessionId(v string)`

SetSdkSessionId sets SdkSessionId field to given value.

### HasSdkSessionId

`func (o *Session) HasSdkSessionId() bool`

HasSdkSessionId returns a boolean if a field has been set.

### GetSdkRestartCount

`func (o *Session) GetSdkRestartCount() int32`

GetSdkRestartCount returns the SdkRestartCount field if non-nil, zero value otherwise.

### GetSdkRestartCountOk

`func (o *Session) GetSdkRestartCountOk() (*int32, bool)`

GetSdkRestartCountOk returns a tuple with the SdkRestartCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSdkRestartCount

`func (o *Session) SetSdkRestartCount(v int32)`

SetSdkRestartCount sets SdkRestartCount field to given value.

### HasSdkRestartCount

`func (o *Session) HasSdkRestartCount() bool`

HasSdkRestartCount returns a boolean if a field has been set.

### GetConditions

`func (o *Session) GetConditions() string`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *Session) GetConditionsOk() (*string, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *Session) SetConditions(v string)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *Session) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetReconciledRepos

`func (o *Session) GetReconciledRepos() string`

GetReconciledRepos returns the ReconciledRepos field if non-nil, zero value otherwise.

### GetReconciledReposOk

`func (o *Session) GetReconciledReposOk() (*string, bool)`

GetReconciledReposOk returns a tuple with the ReconciledRepos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReconciledRepos

`func (o *Session) SetReconciledRepos(v string)`

SetReconciledRepos sets ReconciledRepos field to given value.

### HasReconciledRepos

`func (o *Session) HasReconciledRepos() bool`

HasReconciledRepos returns a boolean if a field has been set.

### GetReconciledWorkflow

`func (o *Session) GetReconciledWorkflow() string`

GetReconciledWorkflow returns the ReconciledWorkflow field if non-nil, zero value otherwise.

### GetReconciledWorkflowOk

`func (o *Session) GetReconciledWorkflowOk() (*string, bool)`

GetReconciledWorkflowOk returns a tuple with the ReconciledWorkflow field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReconciledWorkflow

`func (o *Session) SetReconciledWorkflow(v string)`

SetReconciledWorkflow sets ReconciledWorkflow field to given value.

### HasReconciledWorkflow

`func (o *Session) HasReconciledWorkflow() bool`

HasReconciledWorkflow returns a boolean if a field has been set.

### GetKubeCrName

`func (o *Session) GetKubeCrName() string`

GetKubeCrName returns the KubeCrName field if non-nil, zero value otherwise.

### GetKubeCrNameOk

`func (o *Session) GetKubeCrNameOk() (*string, bool)`

GetKubeCrNameOk returns a tuple with the KubeCrName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubeCrName

`func (o *Session) SetKubeCrName(v string)`

SetKubeCrName sets KubeCrName field to given value.

### HasKubeCrName

`func (o *Session) HasKubeCrName() bool`

HasKubeCrName returns a boolean if a field has been set.

### GetKubeCrUid

`func (o *Session) GetKubeCrUid() string`

GetKubeCrUid returns the KubeCrUid field if non-nil, zero value otherwise.

### GetKubeCrUidOk

`func (o *Session) GetKubeCrUidOk() (*string, bool)`

GetKubeCrUidOk returns a tuple with the KubeCrUid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubeCrUid

`func (o *Session) SetKubeCrUid(v string)`

SetKubeCrUid sets KubeCrUid field to given value.

### HasKubeCrUid

`func (o *Session) HasKubeCrUid() bool`

HasKubeCrUid returns a boolean if a field has been set.

### GetKubeNamespace

`func (o *Session) GetKubeNamespace() string`

GetKubeNamespace returns the KubeNamespace field if non-nil, zero value otherwise.

### GetKubeNamespaceOk

`func (o *Session) GetKubeNamespaceOk() (*string, bool)`

GetKubeNamespaceOk returns a tuple with the KubeNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubeNamespace

`func (o *Session) SetKubeNamespace(v string)`

SetKubeNamespace sets KubeNamespace field to given value.

### HasKubeNamespace

`func (o *Session) HasKubeNamespace() bool`

HasKubeNamespace returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


