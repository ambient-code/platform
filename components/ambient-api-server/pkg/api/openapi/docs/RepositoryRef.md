# RepositoryRef

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Name** | **string** |  | 
**Url** | **string** |  | 
**Branch** | Pointer to **string** |  | [optional] 
**Provider** | Pointer to **string** |  | [optional] 
**Owner** | Pointer to **string** |  | [optional] 
**RepoName** | Pointer to **string** |  | [optional] 
**ProjectId** | Pointer to **string** |  | [optional] 

## Methods

### NewRepositoryRef

`func NewRepositoryRef(name string, url string, ) *RepositoryRef`

NewRepositoryRef instantiates a new RepositoryRef object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRepositoryRefWithDefaults

`func NewRepositoryRefWithDefaults() *RepositoryRef`

NewRepositoryRefWithDefaults instantiates a new RepositoryRef object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *RepositoryRef) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RepositoryRef) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RepositoryRef) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *RepositoryRef) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *RepositoryRef) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *RepositoryRef) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *RepositoryRef) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *RepositoryRef) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *RepositoryRef) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *RepositoryRef) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *RepositoryRef) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *RepositoryRef) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *RepositoryRef) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *RepositoryRef) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *RepositoryRef) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *RepositoryRef) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *RepositoryRef) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *RepositoryRef) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *RepositoryRef) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *RepositoryRef) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetName

`func (o *RepositoryRef) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RepositoryRef) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RepositoryRef) SetName(v string)`

SetName sets Name field to given value.


### GetUrl

`func (o *RepositoryRef) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *RepositoryRef) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *RepositoryRef) SetUrl(v string)`

SetUrl sets Url field to given value.


### GetBranch

`func (o *RepositoryRef) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *RepositoryRef) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *RepositoryRef) SetBranch(v string)`

SetBranch sets Branch field to given value.

### HasBranch

`func (o *RepositoryRef) HasBranch() bool`

HasBranch returns a boolean if a field has been set.

### GetProvider

`func (o *RepositoryRef) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *RepositoryRef) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *RepositoryRef) SetProvider(v string)`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *RepositoryRef) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### GetOwner

`func (o *RepositoryRef) GetOwner() string`

GetOwner returns the Owner field if non-nil, zero value otherwise.

### GetOwnerOk

`func (o *RepositoryRef) GetOwnerOk() (*string, bool)`

GetOwnerOk returns a tuple with the Owner field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwner

`func (o *RepositoryRef) SetOwner(v string)`

SetOwner sets Owner field to given value.

### HasOwner

`func (o *RepositoryRef) HasOwner() bool`

HasOwner returns a boolean if a field has been set.

### GetRepoName

`func (o *RepositoryRef) GetRepoName() string`

GetRepoName returns the RepoName field if non-nil, zero value otherwise.

### GetRepoNameOk

`func (o *RepositoryRef) GetRepoNameOk() (*string, bool)`

GetRepoNameOk returns a tuple with the RepoName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepoName

`func (o *RepositoryRef) SetRepoName(v string)`

SetRepoName sets RepoName field to given value.

### HasRepoName

`func (o *RepositoryRef) HasRepoName() bool`

HasRepoName returns a boolean if a field has been set.

### GetProjectId

`func (o *RepositoryRef) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *RepositoryRef) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *RepositoryRef) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *RepositoryRef) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


