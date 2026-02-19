# Permission

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**SubjectType** | **string** |  | 
**SubjectName** | **string** |  | 
**Role** | **string** |  | 
**ProjectId** | Pointer to **string** |  | [optional] 

## Methods

### NewPermission

`func NewPermission(subjectType string, subjectName string, role string, ) *Permission`

NewPermission instantiates a new Permission object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPermissionWithDefaults

`func NewPermissionWithDefaults() *Permission`

NewPermissionWithDefaults instantiates a new Permission object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Permission) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Permission) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Permission) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Permission) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Permission) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Permission) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Permission) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Permission) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Permission) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Permission) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Permission) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Permission) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Permission) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Permission) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Permission) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Permission) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Permission) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Permission) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Permission) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Permission) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetSubjectType

`func (o *Permission) GetSubjectType() string`

GetSubjectType returns the SubjectType field if non-nil, zero value otherwise.

### GetSubjectTypeOk

`func (o *Permission) GetSubjectTypeOk() (*string, bool)`

GetSubjectTypeOk returns a tuple with the SubjectType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubjectType

`func (o *Permission) SetSubjectType(v string)`

SetSubjectType sets SubjectType field to given value.


### GetSubjectName

`func (o *Permission) GetSubjectName() string`

GetSubjectName returns the SubjectName field if non-nil, zero value otherwise.

### GetSubjectNameOk

`func (o *Permission) GetSubjectNameOk() (*string, bool)`

GetSubjectNameOk returns a tuple with the SubjectName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSubjectName

`func (o *Permission) SetSubjectName(v string)`

SetSubjectName sets SubjectName field to given value.


### GetRole

`func (o *Permission) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *Permission) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *Permission) SetRole(v string)`

SetRole sets Role field to given value.


### GetProjectId

`func (o *Permission) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *Permission) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *Permission) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.

### HasProjectId

`func (o *Permission) HasProjectId() bool`

HasProjectId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


