# ProjectSettings

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**ProjectId** | **string** |  | 
**GroupAccess** | Pointer to **string** |  | [optional] 
**RunnerSecrets** | Pointer to **string** |  | [optional] 
**Repositories** | Pointer to **string** |  | [optional] 

## Methods

### NewProjectSettings

`func NewProjectSettings(projectId string, ) *ProjectSettings`

NewProjectSettings instantiates a new ProjectSettings object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectSettingsWithDefaults

`func NewProjectSettingsWithDefaults() *ProjectSettings`

NewProjectSettingsWithDefaults instantiates a new ProjectSettings object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *ProjectSettings) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *ProjectSettings) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *ProjectSettings) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *ProjectSettings) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *ProjectSettings) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *ProjectSettings) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *ProjectSettings) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *ProjectSettings) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *ProjectSettings) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *ProjectSettings) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *ProjectSettings) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *ProjectSettings) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *ProjectSettings) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *ProjectSettings) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *ProjectSettings) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *ProjectSettings) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *ProjectSettings) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *ProjectSettings) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *ProjectSettings) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *ProjectSettings) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetProjectId

`func (o *ProjectSettings) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *ProjectSettings) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *ProjectSettings) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetGroupAccess

`func (o *ProjectSettings) GetGroupAccess() string`

GetGroupAccess returns the GroupAccess field if non-nil, zero value otherwise.

### GetGroupAccessOk

`func (o *ProjectSettings) GetGroupAccessOk() (*string, bool)`

GetGroupAccessOk returns a tuple with the GroupAccess field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroupAccess

`func (o *ProjectSettings) SetGroupAccess(v string)`

SetGroupAccess sets GroupAccess field to given value.

### HasGroupAccess

`func (o *ProjectSettings) HasGroupAccess() bool`

HasGroupAccess returns a boolean if a field has been set.

### GetRunnerSecrets

`func (o *ProjectSettings) GetRunnerSecrets() string`

GetRunnerSecrets returns the RunnerSecrets field if non-nil, zero value otherwise.

### GetRunnerSecretsOk

`func (o *ProjectSettings) GetRunnerSecretsOk() (*string, bool)`

GetRunnerSecretsOk returns a tuple with the RunnerSecrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunnerSecrets

`func (o *ProjectSettings) SetRunnerSecrets(v string)`

SetRunnerSecrets sets RunnerSecrets field to given value.

### HasRunnerSecrets

`func (o *ProjectSettings) HasRunnerSecrets() bool`

HasRunnerSecrets returns a boolean if a field has been set.

### GetRepositories

`func (o *ProjectSettings) GetRepositories() string`

GetRepositories returns the Repositories field if non-nil, zero value otherwise.

### GetRepositoriesOk

`func (o *ProjectSettings) GetRepositoriesOk() (*string, bool)`

GetRepositoriesOk returns a tuple with the Repositories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepositories

`func (o *ProjectSettings) SetRepositories(v string)`

SetRepositories sets Repositories field to given value.

### HasRepositories

`func (o *ProjectSettings) HasRepositories() bool`

HasRepositories returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


