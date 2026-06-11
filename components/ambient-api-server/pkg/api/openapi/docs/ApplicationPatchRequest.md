# ApplicationPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**SourceRepoUrl** | Pointer to **string** |  | [optional] 
**SourceTargetRevision** | Pointer to **string** |  | [optional] 
**SourcePath** | Pointer to **string** |  | [optional] 
**DestinationAmbientUrl** | Pointer to **string** |  | [optional] 
**DestinationProject** | Pointer to **string** |  | [optional] 
**CredentialId** | Pointer to **string** |  | [optional] 
**AutoSync** | Pointer to **bool** |  | [optional] 
**AutoPrune** | Pointer to **bool** |  | [optional] 
**SelfHeal** | Pointer to **bool** |  | [optional] 
**SyncOptions** | Pointer to **string** |  | [optional] 
**RetryLimit** | Pointer to **int32** |  | [optional] 
**SyncStatus** | Pointer to **string** |  | [optional] 
**HealthStatus** | Pointer to **string** |  | [optional] 
**SyncRevision** | Pointer to **string** |  | [optional] 
**OperationPhase** | Pointer to **string** |  | [optional] 
**OperationMessage** | Pointer to **string** |  | [optional] 
**ResourceStatus** | Pointer to **string** |  | [optional] 
**Conditions** | Pointer to **string** |  | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 
**LastSyncedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewApplicationPatchRequest

`func NewApplicationPatchRequest() *ApplicationPatchRequest`

NewApplicationPatchRequest instantiates a new ApplicationPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApplicationPatchRequestWithDefaults

`func NewApplicationPatchRequestWithDefaults() *ApplicationPatchRequest`

NewApplicationPatchRequestWithDefaults instantiates a new ApplicationPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ApplicationPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ApplicationPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ApplicationPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ApplicationPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetSourceRepoUrl

`func (o *ApplicationPatchRequest) GetSourceRepoUrl() string`

GetSourceRepoUrl returns the SourceRepoUrl field if non-nil, zero value otherwise.

### GetSourceRepoUrlOk

`func (o *ApplicationPatchRequest) GetSourceRepoUrlOk() (*string, bool)`

GetSourceRepoUrlOk returns a tuple with the SourceRepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourceRepoUrl

`func (o *ApplicationPatchRequest) SetSourceRepoUrl(v string)`

SetSourceRepoUrl sets SourceRepoUrl field to given value.

### HasSourceRepoUrl

`func (o *ApplicationPatchRequest) HasSourceRepoUrl() bool`

HasSourceRepoUrl returns a boolean if a field has been set.

### GetSourceTargetRevision

`func (o *ApplicationPatchRequest) GetSourceTargetRevision() string`

GetSourceTargetRevision returns the SourceTargetRevision field if non-nil, zero value otherwise.

### GetSourceTargetRevisionOk

`func (o *ApplicationPatchRequest) GetSourceTargetRevisionOk() (*string, bool)`

GetSourceTargetRevisionOk returns a tuple with the SourceTargetRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourceTargetRevision

`func (o *ApplicationPatchRequest) SetSourceTargetRevision(v string)`

SetSourceTargetRevision sets SourceTargetRevision field to given value.

### HasSourceTargetRevision

`func (o *ApplicationPatchRequest) HasSourceTargetRevision() bool`

HasSourceTargetRevision returns a boolean if a field has been set.

### GetSourcePath

`func (o *ApplicationPatchRequest) GetSourcePath() string`

GetSourcePath returns the SourcePath field if non-nil, zero value otherwise.

### GetSourcePathOk

`func (o *ApplicationPatchRequest) GetSourcePathOk() (*string, bool)`

GetSourcePathOk returns a tuple with the SourcePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourcePath

`func (o *ApplicationPatchRequest) SetSourcePath(v string)`

SetSourcePath sets SourcePath field to given value.

### HasSourcePath

`func (o *ApplicationPatchRequest) HasSourcePath() bool`

HasSourcePath returns a boolean if a field has been set.

### GetDestinationAmbientUrl

`func (o *ApplicationPatchRequest) GetDestinationAmbientUrl() string`

GetDestinationAmbientUrl returns the DestinationAmbientUrl field if non-nil, zero value otherwise.

### GetDestinationAmbientUrlOk

`func (o *ApplicationPatchRequest) GetDestinationAmbientUrlOk() (*string, bool)`

GetDestinationAmbientUrlOk returns a tuple with the DestinationAmbientUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDestinationAmbientUrl

`func (o *ApplicationPatchRequest) SetDestinationAmbientUrl(v string)`

SetDestinationAmbientUrl sets DestinationAmbientUrl field to given value.

### HasDestinationAmbientUrl

`func (o *ApplicationPatchRequest) HasDestinationAmbientUrl() bool`

HasDestinationAmbientUrl returns a boolean if a field has been set.

### GetDestinationProject

`func (o *ApplicationPatchRequest) GetDestinationProject() string`

GetDestinationProject returns the DestinationProject field if non-nil, zero value otherwise.

### GetDestinationProjectOk

`func (o *ApplicationPatchRequest) GetDestinationProjectOk() (*string, bool)`

GetDestinationProjectOk returns a tuple with the DestinationProject field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDestinationProject

`func (o *ApplicationPatchRequest) SetDestinationProject(v string)`

SetDestinationProject sets DestinationProject field to given value.

### HasDestinationProject

`func (o *ApplicationPatchRequest) HasDestinationProject() bool`

HasDestinationProject returns a boolean if a field has been set.

### GetCredentialId

`func (o *ApplicationPatchRequest) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *ApplicationPatchRequest) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *ApplicationPatchRequest) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.

### HasCredentialId

`func (o *ApplicationPatchRequest) HasCredentialId() bool`

HasCredentialId returns a boolean if a field has been set.

### GetAutoSync

`func (o *ApplicationPatchRequest) GetAutoSync() bool`

GetAutoSync returns the AutoSync field if non-nil, zero value otherwise.

### GetAutoSyncOk

`func (o *ApplicationPatchRequest) GetAutoSyncOk() (*bool, bool)`

GetAutoSyncOk returns a tuple with the AutoSync field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoSync

`func (o *ApplicationPatchRequest) SetAutoSync(v bool)`

SetAutoSync sets AutoSync field to given value.

### HasAutoSync

`func (o *ApplicationPatchRequest) HasAutoSync() bool`

HasAutoSync returns a boolean if a field has been set.

### GetAutoPrune

`func (o *ApplicationPatchRequest) GetAutoPrune() bool`

GetAutoPrune returns the AutoPrune field if non-nil, zero value otherwise.

### GetAutoPruneOk

`func (o *ApplicationPatchRequest) GetAutoPruneOk() (*bool, bool)`

GetAutoPruneOk returns a tuple with the AutoPrune field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPrune

`func (o *ApplicationPatchRequest) SetAutoPrune(v bool)`

SetAutoPrune sets AutoPrune field to given value.

### HasAutoPrune

`func (o *ApplicationPatchRequest) HasAutoPrune() bool`

HasAutoPrune returns a boolean if a field has been set.

### GetSelfHeal

`func (o *ApplicationPatchRequest) GetSelfHeal() bool`

GetSelfHeal returns the SelfHeal field if non-nil, zero value otherwise.

### GetSelfHealOk

`func (o *ApplicationPatchRequest) GetSelfHealOk() (*bool, bool)`

GetSelfHealOk returns a tuple with the SelfHeal field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelfHeal

`func (o *ApplicationPatchRequest) SetSelfHeal(v bool)`

SetSelfHeal sets SelfHeal field to given value.

### HasSelfHeal

`func (o *ApplicationPatchRequest) HasSelfHeal() bool`

HasSelfHeal returns a boolean if a field has been set.

### GetSyncOptions

`func (o *ApplicationPatchRequest) GetSyncOptions() string`

GetSyncOptions returns the SyncOptions field if non-nil, zero value otherwise.

### GetSyncOptionsOk

`func (o *ApplicationPatchRequest) GetSyncOptionsOk() (*string, bool)`

GetSyncOptionsOk returns a tuple with the SyncOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncOptions

`func (o *ApplicationPatchRequest) SetSyncOptions(v string)`

SetSyncOptions sets SyncOptions field to given value.

### HasSyncOptions

`func (o *ApplicationPatchRequest) HasSyncOptions() bool`

HasSyncOptions returns a boolean if a field has been set.

### GetRetryLimit

`func (o *ApplicationPatchRequest) GetRetryLimit() int32`

GetRetryLimit returns the RetryLimit field if non-nil, zero value otherwise.

### GetRetryLimitOk

`func (o *ApplicationPatchRequest) GetRetryLimitOk() (*int32, bool)`

GetRetryLimitOk returns a tuple with the RetryLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRetryLimit

`func (o *ApplicationPatchRequest) SetRetryLimit(v int32)`

SetRetryLimit sets RetryLimit field to given value.

### HasRetryLimit

`func (o *ApplicationPatchRequest) HasRetryLimit() bool`

HasRetryLimit returns a boolean if a field has been set.

### GetSyncStatus

`func (o *ApplicationPatchRequest) GetSyncStatus() string`

GetSyncStatus returns the SyncStatus field if non-nil, zero value otherwise.

### GetSyncStatusOk

`func (o *ApplicationPatchRequest) GetSyncStatusOk() (*string, bool)`

GetSyncStatusOk returns a tuple with the SyncStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncStatus

`func (o *ApplicationPatchRequest) SetSyncStatus(v string)`

SetSyncStatus sets SyncStatus field to given value.

### HasSyncStatus

`func (o *ApplicationPatchRequest) HasSyncStatus() bool`

HasSyncStatus returns a boolean if a field has been set.

### GetHealthStatus

`func (o *ApplicationPatchRequest) GetHealthStatus() string`

GetHealthStatus returns the HealthStatus field if non-nil, zero value otherwise.

### GetHealthStatusOk

`func (o *ApplicationPatchRequest) GetHealthStatusOk() (*string, bool)`

GetHealthStatusOk returns a tuple with the HealthStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealthStatus

`func (o *ApplicationPatchRequest) SetHealthStatus(v string)`

SetHealthStatus sets HealthStatus field to given value.

### HasHealthStatus

`func (o *ApplicationPatchRequest) HasHealthStatus() bool`

HasHealthStatus returns a boolean if a field has been set.

### GetSyncRevision

`func (o *ApplicationPatchRequest) GetSyncRevision() string`

GetSyncRevision returns the SyncRevision field if non-nil, zero value otherwise.

### GetSyncRevisionOk

`func (o *ApplicationPatchRequest) GetSyncRevisionOk() (*string, bool)`

GetSyncRevisionOk returns a tuple with the SyncRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncRevision

`func (o *ApplicationPatchRequest) SetSyncRevision(v string)`

SetSyncRevision sets SyncRevision field to given value.

### HasSyncRevision

`func (o *ApplicationPatchRequest) HasSyncRevision() bool`

HasSyncRevision returns a boolean if a field has been set.

### GetOperationPhase

`func (o *ApplicationPatchRequest) GetOperationPhase() string`

GetOperationPhase returns the OperationPhase field if non-nil, zero value otherwise.

### GetOperationPhaseOk

`func (o *ApplicationPatchRequest) GetOperationPhaseOk() (*string, bool)`

GetOperationPhaseOk returns a tuple with the OperationPhase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperationPhase

`func (o *ApplicationPatchRequest) SetOperationPhase(v string)`

SetOperationPhase sets OperationPhase field to given value.

### HasOperationPhase

`func (o *ApplicationPatchRequest) HasOperationPhase() bool`

HasOperationPhase returns a boolean if a field has been set.

### GetOperationMessage

`func (o *ApplicationPatchRequest) GetOperationMessage() string`

GetOperationMessage returns the OperationMessage field if non-nil, zero value otherwise.

### GetOperationMessageOk

`func (o *ApplicationPatchRequest) GetOperationMessageOk() (*string, bool)`

GetOperationMessageOk returns a tuple with the OperationMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperationMessage

`func (o *ApplicationPatchRequest) SetOperationMessage(v string)`

SetOperationMessage sets OperationMessage field to given value.

### HasOperationMessage

`func (o *ApplicationPatchRequest) HasOperationMessage() bool`

HasOperationMessage returns a boolean if a field has been set.

### GetResourceStatus

`func (o *ApplicationPatchRequest) GetResourceStatus() string`

GetResourceStatus returns the ResourceStatus field if non-nil, zero value otherwise.

### GetResourceStatusOk

`func (o *ApplicationPatchRequest) GetResourceStatusOk() (*string, bool)`

GetResourceStatusOk returns a tuple with the ResourceStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceStatus

`func (o *ApplicationPatchRequest) SetResourceStatus(v string)`

SetResourceStatus sets ResourceStatus field to given value.

### HasResourceStatus

`func (o *ApplicationPatchRequest) HasResourceStatus() bool`

HasResourceStatus returns a boolean if a field has been set.

### GetConditions

`func (o *ApplicationPatchRequest) GetConditions() string`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *ApplicationPatchRequest) GetConditionsOk() (*string, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *ApplicationPatchRequest) SetConditions(v string)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *ApplicationPatchRequest) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetLabels

`func (o *ApplicationPatchRequest) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *ApplicationPatchRequest) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *ApplicationPatchRequest) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *ApplicationPatchRequest) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *ApplicationPatchRequest) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *ApplicationPatchRequest) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *ApplicationPatchRequest) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *ApplicationPatchRequest) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetLastSyncedAt

`func (o *ApplicationPatchRequest) GetLastSyncedAt() time.Time`

GetLastSyncedAt returns the LastSyncedAt field if non-nil, zero value otherwise.

### GetLastSyncedAtOk

`func (o *ApplicationPatchRequest) GetLastSyncedAtOk() (*time.Time, bool)`

GetLastSyncedAtOk returns a tuple with the LastSyncedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSyncedAt

`func (o *ApplicationPatchRequest) SetLastSyncedAt(v time.Time)`

SetLastSyncedAt sets LastSyncedAt field to given value.

### HasLastSyncedAt

`func (o *ApplicationPatchRequest) HasLastSyncedAt() bool`

HasLastSyncedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


