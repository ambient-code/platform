# ProjectSettingsPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ProjectId** | Pointer to **string** |  | [optional] 
**GroupAccess** | Pointer to **string** |  | [optional] 
**Repositories** | Pointer to **string** |  | [optional] 

## Methods

### NewProjectSettingsPatchRequest

`func NewProjectSettingsPatchRequest() *ProjectSettingsPatchRequest`

NewProjectSettingsPatchRequest instantiates a new ProjectSettingsPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectSettingsPatchRequestWithDefaults

`func NewProjectSettingsPatchRequestWithDefaults() *ProjectSettingsPatchRequest`

NewProjectSettingsPatchRequestWithDefaults instantiates a new ProjectSettingsPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProjectId

`func (o *ProjectSettingsPatchRequest) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *ProjectSettingsPatchRequest) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *ProjectSettingsPatchRequest) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *ProjectSettingsPatchRequest) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.

### GetGroupAccess

`func (o *ProjectSettingsPatchRequest) GetGroupAccess() string`

GetGroupAccess returns the GroupAccess field if non-nil, zero value otherwise.

### GetGroupAccessOk

`func (o *ProjectSettingsPatchRequest) GetGroupAccessOk() (*string, bool)`

GetGroupAccessOk returns a tuple with the GroupAccess field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroupAccess

`func (o *ProjectSettingsPatchRequest) SetGroupAccess(v string)`

SetGroupAccess sets GroupAccess field to given value.

### HasGroupAccess

`func (o *ProjectSettingsPatchRequest) HasGroupAccess() bool`

HasGroupAccess returns a boolean if a field has been set.

### GetRepositories

`func (o *ProjectSettingsPatchRequest) GetRepositories() string`

GetRepositories returns the Repositories field if non-nil, zero value otherwise.

### GetRepositoriesOk

`func (o *ProjectSettingsPatchRequest) GetRepositoriesOk() (*string, bool)`

GetRepositoriesOk returns a tuple with the Repositories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepositories

`func (o *ProjectSettingsPatchRequest) SetRepositories(v string)`

SetRepositories sets Repositories field to given value.

### HasRepositories

`func (o *ProjectSettingsPatchRequest) HasRepositories() bool`

HasRepositories returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


