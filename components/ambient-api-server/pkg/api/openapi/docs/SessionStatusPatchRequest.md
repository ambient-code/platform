# SessionStatusPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Phase** | Pointer to **string** |  | [optional] 
**StartTime** | Pointer to **time.Time** |  | [optional] 
**CompletionTime** | Pointer to **time.Time** |  | [optional] 
**SdkSessionId** | Pointer to **string** |  | [optional] 
**SdkRestartCount** | Pointer to **int32** |  | [optional] 
**Conditions** | Pointer to **string** |  | [optional] 
**ReconciledRepos** | Pointer to **string** |  | [optional] 
**ReconciledWorkflow** | Pointer to **string** |  | [optional] 
**KubeCrUid** | Pointer to **string** |  | [optional] 
**KubeNamespace** | Pointer to **string** |  | [optional] 

## Methods

### NewSessionStatusPatchRequest

`func NewSessionStatusPatchRequest() *SessionStatusPatchRequest`

NewSessionStatusPatchRequest instantiates a new SessionStatusPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionStatusPatchRequestWithDefaults

`func NewSessionStatusPatchRequestWithDefaults() *SessionStatusPatchRequest`

NewSessionStatusPatchRequestWithDefaults instantiates a new SessionStatusPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPhase

`func (o *SessionStatusPatchRequest) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *SessionStatusPatchRequest) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *SessionStatusPatchRequest) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *SessionStatusPatchRequest) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStartTime

`func (o *SessionStatusPatchRequest) GetStartTime() time.Time`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *SessionStatusPatchRequest) GetStartTimeOk() (*time.Time, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *SessionStatusPatchRequest) SetStartTime(v time.Time)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *SessionStatusPatchRequest) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.

### GetCompletionTime

`func (o *SessionStatusPatchRequest) GetCompletionTime() time.Time`

GetCompletionTime returns the CompletionTime field if non-nil, zero value otherwise.

### GetCompletionTimeOk

`func (o *SessionStatusPatchRequest) GetCompletionTimeOk() (*time.Time, bool)`

GetCompletionTimeOk returns a tuple with the CompletionTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletionTime

`func (o *SessionStatusPatchRequest) SetCompletionTime(v time.Time)`

SetCompletionTime sets CompletionTime field to given value.

### HasCompletionTime

`func (o *SessionStatusPatchRequest) HasCompletionTime() bool`

HasCompletionTime returns a boolean if a field has been set.

### GetSdkSessionId

`func (o *SessionStatusPatchRequest) GetSdkSessionId() string`

GetSdkSessionId returns the SdkSessionId field if non-nil, zero value otherwise.

### GetSdkSessionIdOk

`func (o *SessionStatusPatchRequest) GetSdkSessionIdOk() (*string, bool)`

GetSdkSessionIdOk returns a tuple with the SdkSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSdkSessionId

`func (o *SessionStatusPatchRequest) SetSdkSessionId(v string)`

SetSdkSessionId sets SdkSessionId field to given value.

### HasSdkSessionId

`func (o *SessionStatusPatchRequest) HasSdkSessionId() bool`

HasSdkSessionId returns a boolean if a field has been set.

### GetSdkRestartCount

`func (o *SessionStatusPatchRequest) GetSdkRestartCount() int32`

GetSdkRestartCount returns the SdkRestartCount field if non-nil, zero value otherwise.

### GetSdkRestartCountOk

`func (o *SessionStatusPatchRequest) GetSdkRestartCountOk() (*int32, bool)`

GetSdkRestartCountOk returns a tuple with the SdkRestartCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSdkRestartCount

`func (o *SessionStatusPatchRequest) SetSdkRestartCount(v int32)`

SetSdkRestartCount sets SdkRestartCount field to given value.

### HasSdkRestartCount

`func (o *SessionStatusPatchRequest) HasSdkRestartCount() bool`

HasSdkRestartCount returns a boolean if a field has been set.

### GetConditions

`func (o *SessionStatusPatchRequest) GetConditions() string`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *SessionStatusPatchRequest) GetConditionsOk() (*string, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *SessionStatusPatchRequest) SetConditions(v string)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *SessionStatusPatchRequest) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetReconciledRepos

`func (o *SessionStatusPatchRequest) GetReconciledRepos() string`

GetReconciledRepos returns the ReconciledRepos field if non-nil, zero value otherwise.

### GetReconciledReposOk

`func (o *SessionStatusPatchRequest) GetReconciledReposOk() (*string, bool)`

GetReconciledReposOk returns a tuple with the ReconciledRepos field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReconciledRepos

`func (o *SessionStatusPatchRequest) SetReconciledRepos(v string)`

SetReconciledRepos sets ReconciledRepos field to given value.

### HasReconciledRepos

`func (o *SessionStatusPatchRequest) HasReconciledRepos() bool`

HasReconciledRepos returns a boolean if a field has been set.

### GetReconciledWorkflow

`func (o *SessionStatusPatchRequest) GetReconciledWorkflow() string`

GetReconciledWorkflow returns the ReconciledWorkflow field if non-nil, zero value otherwise.

### GetReconciledWorkflowOk

`func (o *SessionStatusPatchRequest) GetReconciledWorkflowOk() (*string, bool)`

GetReconciledWorkflowOk returns a tuple with the ReconciledWorkflow field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReconciledWorkflow

`func (o *SessionStatusPatchRequest) SetReconciledWorkflow(v string)`

SetReconciledWorkflow sets ReconciledWorkflow field to given value.

### HasReconciledWorkflow

`func (o *SessionStatusPatchRequest) HasReconciledWorkflow() bool`

HasReconciledWorkflow returns a boolean if a field has been set.

### GetKubeCrUid

`func (o *SessionStatusPatchRequest) GetKubeCrUid() string`

GetKubeCrUid returns the KubeCrUid field if non-nil, zero value otherwise.

### GetKubeCrUidOk

`func (o *SessionStatusPatchRequest) GetKubeCrUidOk() (*string, bool)`

GetKubeCrUidOk returns a tuple with the KubeCrUid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubeCrUid

`func (o *SessionStatusPatchRequest) SetKubeCrUid(v string)`

SetKubeCrUid sets KubeCrUid field to given value.

### HasKubeCrUid

`func (o *SessionStatusPatchRequest) HasKubeCrUid() bool`

HasKubeCrUid returns a boolean if a field has been set.

### GetKubeNamespace

`func (o *SessionStatusPatchRequest) GetKubeNamespace() string`

GetKubeNamespace returns the KubeNamespace field if non-nil, zero value otherwise.

### GetKubeNamespaceOk

`func (o *SessionStatusPatchRequest) GetKubeNamespaceOk() (*string, bool)`

GetKubeNamespaceOk returns a tuple with the KubeNamespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKubeNamespace

`func (o *SessionStatusPatchRequest) SetKubeNamespace(v string)`

SetKubeNamespace sets KubeNamespace field to given value.

### HasKubeNamespace

`func (o *SessionStatusPatchRequest) HasKubeNamespace() bool`

HasKubeNamespace returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


