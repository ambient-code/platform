# Application

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Name** | **string** |  | 
**SourceRepoUrl** | **string** |  | 
**SourceTargetRevision** | Pointer to **string** |  | [optional] 
**SourcePath** | **string** |  | 
**DestinationAmbientUrl** | Pointer to **string** |  | [optional] 
**DestinationProject** | **string** |  | 
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

### NewApplication

`func NewApplication(name string, sourceRepoUrl string, sourcePath string, destinationProject string, ) *Application`

NewApplication instantiates a new Application object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApplicationWithDefaults

`func NewApplicationWithDefaults() *Application`

NewApplicationWithDefaults instantiates a new Application object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Application) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Application) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Application) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Application) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Application) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Application) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Application) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Application) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Application) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Application) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Application) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Application) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Application) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Application) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Application) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Application) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Application) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Application) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Application) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Application) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetName

`func (o *Application) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Application) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Application) SetName(v string)`

SetName sets Name field to given value.


### GetSourceRepoUrl

`func (o *Application) GetSourceRepoUrl() string`

GetSourceRepoUrl returns the SourceRepoUrl field if non-nil, zero value otherwise.

### GetSourceRepoUrlOk

`func (o *Application) GetSourceRepoUrlOk() (*string, bool)`

GetSourceRepoUrlOk returns a tuple with the SourceRepoUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourceRepoUrl

`func (o *Application) SetSourceRepoUrl(v string)`

SetSourceRepoUrl sets SourceRepoUrl field to given value.


### GetSourceTargetRevision

`func (o *Application) GetSourceTargetRevision() string`

GetSourceTargetRevision returns the SourceTargetRevision field if non-nil, zero value otherwise.

### GetSourceTargetRevisionOk

`func (o *Application) GetSourceTargetRevisionOk() (*string, bool)`

GetSourceTargetRevisionOk returns a tuple with the SourceTargetRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourceTargetRevision

`func (o *Application) SetSourceTargetRevision(v string)`

SetSourceTargetRevision sets SourceTargetRevision field to given value.

### HasSourceTargetRevision

`func (o *Application) HasSourceTargetRevision() bool`

HasSourceTargetRevision returns a boolean if a field has been set.

### GetSourcePath

`func (o *Application) GetSourcePath() string`

GetSourcePath returns the SourcePath field if non-nil, zero value otherwise.

### GetSourcePathOk

`func (o *Application) GetSourcePathOk() (*string, bool)`

GetSourcePathOk returns a tuple with the SourcePath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSourcePath

`func (o *Application) SetSourcePath(v string)`

SetSourcePath sets SourcePath field to given value.


### GetDestinationAmbientUrl

`func (o *Application) GetDestinationAmbientUrl() string`

GetDestinationAmbientUrl returns the DestinationAmbientUrl field if non-nil, zero value otherwise.

### GetDestinationAmbientUrlOk

`func (o *Application) GetDestinationAmbientUrlOk() (*string, bool)`

GetDestinationAmbientUrlOk returns a tuple with the DestinationAmbientUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDestinationAmbientUrl

`func (o *Application) SetDestinationAmbientUrl(v string)`

SetDestinationAmbientUrl sets DestinationAmbientUrl field to given value.

### HasDestinationAmbientUrl

`func (o *Application) HasDestinationAmbientUrl() bool`

HasDestinationAmbientUrl returns a boolean if a field has been set.

### GetDestinationProject

`func (o *Application) GetDestinationProject() string`

GetDestinationProject returns the DestinationProject field if non-nil, zero value otherwise.

### GetDestinationProjectOk

`func (o *Application) GetDestinationProjectOk() (*string, bool)`

GetDestinationProjectOk returns a tuple with the DestinationProject field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDestinationProject

`func (o *Application) SetDestinationProject(v string)`

SetDestinationProject sets DestinationProject field to given value.


### GetCredentialId

`func (o *Application) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *Application) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *Application) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.

### HasCredentialId

`func (o *Application) HasCredentialId() bool`

HasCredentialId returns a boolean if a field has been set.

### GetAutoSync

`func (o *Application) GetAutoSync() bool`

GetAutoSync returns the AutoSync field if non-nil, zero value otherwise.

### GetAutoSyncOk

`func (o *Application) GetAutoSyncOk() (*bool, bool)`

GetAutoSyncOk returns a tuple with the AutoSync field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoSync

`func (o *Application) SetAutoSync(v bool)`

SetAutoSync sets AutoSync field to given value.

### HasAutoSync

`func (o *Application) HasAutoSync() bool`

HasAutoSync returns a boolean if a field has been set.

### GetAutoPrune

`func (o *Application) GetAutoPrune() bool`

GetAutoPrune returns the AutoPrune field if non-nil, zero value otherwise.

### GetAutoPruneOk

`func (o *Application) GetAutoPruneOk() (*bool, bool)`

GetAutoPruneOk returns a tuple with the AutoPrune field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoPrune

`func (o *Application) SetAutoPrune(v bool)`

SetAutoPrune sets AutoPrune field to given value.

### HasAutoPrune

`func (o *Application) HasAutoPrune() bool`

HasAutoPrune returns a boolean if a field has been set.

### GetSelfHeal

`func (o *Application) GetSelfHeal() bool`

GetSelfHeal returns the SelfHeal field if non-nil, zero value otherwise.

### GetSelfHealOk

`func (o *Application) GetSelfHealOk() (*bool, bool)`

GetSelfHealOk returns a tuple with the SelfHeal field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelfHeal

`func (o *Application) SetSelfHeal(v bool)`

SetSelfHeal sets SelfHeal field to given value.

### HasSelfHeal

`func (o *Application) HasSelfHeal() bool`

HasSelfHeal returns a boolean if a field has been set.

### GetSyncOptions

`func (o *Application) GetSyncOptions() string`

GetSyncOptions returns the SyncOptions field if non-nil, zero value otherwise.

### GetSyncOptionsOk

`func (o *Application) GetSyncOptionsOk() (*string, bool)`

GetSyncOptionsOk returns a tuple with the SyncOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncOptions

`func (o *Application) SetSyncOptions(v string)`

SetSyncOptions sets SyncOptions field to given value.

### HasSyncOptions

`func (o *Application) HasSyncOptions() bool`

HasSyncOptions returns a boolean if a field has been set.

### GetRetryLimit

`func (o *Application) GetRetryLimit() int32`

GetRetryLimit returns the RetryLimit field if non-nil, zero value otherwise.

### GetRetryLimitOk

`func (o *Application) GetRetryLimitOk() (*int32, bool)`

GetRetryLimitOk returns a tuple with the RetryLimit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRetryLimit

`func (o *Application) SetRetryLimit(v int32)`

SetRetryLimit sets RetryLimit field to given value.

### HasRetryLimit

`func (o *Application) HasRetryLimit() bool`

HasRetryLimit returns a boolean if a field has been set.

### GetSyncStatus

`func (o *Application) GetSyncStatus() string`

GetSyncStatus returns the SyncStatus field if non-nil, zero value otherwise.

### GetSyncStatusOk

`func (o *Application) GetSyncStatusOk() (*string, bool)`

GetSyncStatusOk returns a tuple with the SyncStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncStatus

`func (o *Application) SetSyncStatus(v string)`

SetSyncStatus sets SyncStatus field to given value.

### HasSyncStatus

`func (o *Application) HasSyncStatus() bool`

HasSyncStatus returns a boolean if a field has been set.

### GetHealthStatus

`func (o *Application) GetHealthStatus() string`

GetHealthStatus returns the HealthStatus field if non-nil, zero value otherwise.

### GetHealthStatusOk

`func (o *Application) GetHealthStatusOk() (*string, bool)`

GetHealthStatusOk returns a tuple with the HealthStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHealthStatus

`func (o *Application) SetHealthStatus(v string)`

SetHealthStatus sets HealthStatus field to given value.

### HasHealthStatus

`func (o *Application) HasHealthStatus() bool`

HasHealthStatus returns a boolean if a field has been set.

### GetSyncRevision

`func (o *Application) GetSyncRevision() string`

GetSyncRevision returns the SyncRevision field if non-nil, zero value otherwise.

### GetSyncRevisionOk

`func (o *Application) GetSyncRevisionOk() (*string, bool)`

GetSyncRevisionOk returns a tuple with the SyncRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSyncRevision

`func (o *Application) SetSyncRevision(v string)`

SetSyncRevision sets SyncRevision field to given value.

### HasSyncRevision

`func (o *Application) HasSyncRevision() bool`

HasSyncRevision returns a boolean if a field has been set.

### GetOperationPhase

`func (o *Application) GetOperationPhase() string`

GetOperationPhase returns the OperationPhase field if non-nil, zero value otherwise.

### GetOperationPhaseOk

`func (o *Application) GetOperationPhaseOk() (*string, bool)`

GetOperationPhaseOk returns a tuple with the OperationPhase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperationPhase

`func (o *Application) SetOperationPhase(v string)`

SetOperationPhase sets OperationPhase field to given value.

### HasOperationPhase

`func (o *Application) HasOperationPhase() bool`

HasOperationPhase returns a boolean if a field has been set.

### GetOperationMessage

`func (o *Application) GetOperationMessage() string`

GetOperationMessage returns the OperationMessage field if non-nil, zero value otherwise.

### GetOperationMessageOk

`func (o *Application) GetOperationMessageOk() (*string, bool)`

GetOperationMessageOk returns a tuple with the OperationMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperationMessage

`func (o *Application) SetOperationMessage(v string)`

SetOperationMessage sets OperationMessage field to given value.

### HasOperationMessage

`func (o *Application) HasOperationMessage() bool`

HasOperationMessage returns a boolean if a field has been set.

### GetResourceStatus

`func (o *Application) GetResourceStatus() string`

GetResourceStatus returns the ResourceStatus field if non-nil, zero value otherwise.

### GetResourceStatusOk

`func (o *Application) GetResourceStatusOk() (*string, bool)`

GetResourceStatusOk returns a tuple with the ResourceStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceStatus

`func (o *Application) SetResourceStatus(v string)`

SetResourceStatus sets ResourceStatus field to given value.

### HasResourceStatus

`func (o *Application) HasResourceStatus() bool`

HasResourceStatus returns a boolean if a field has been set.

### GetConditions

`func (o *Application) GetConditions() string`

GetConditions returns the Conditions field if non-nil, zero value otherwise.

### GetConditionsOk

`func (o *Application) GetConditionsOk() (*string, bool)`

GetConditionsOk returns a tuple with the Conditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConditions

`func (o *Application) SetConditions(v string)`

SetConditions sets Conditions field to given value.

### HasConditions

`func (o *Application) HasConditions() bool`

HasConditions returns a boolean if a field has been set.

### GetLabels

`func (o *Application) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *Application) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *Application) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *Application) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *Application) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *Application) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *Application) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *Application) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.

### GetLastSyncedAt

`func (o *Application) GetLastSyncedAt() time.Time`

GetLastSyncedAt returns the LastSyncedAt field if non-nil, zero value otherwise.

### GetLastSyncedAtOk

`func (o *Application) GetLastSyncedAtOk() (*time.Time, bool)`

GetLastSyncedAtOk returns a tuple with the LastSyncedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSyncedAt

`func (o *Application) SetLastSyncedAt(v time.Time)`

SetLastSyncedAt sets LastSyncedAt field to given value.

### HasLastSyncedAt

`func (o *Application) HasLastSyncedAt() bool`

HasLastSyncedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


