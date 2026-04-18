# Credential

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**ProjectId** | **string** | ID of the project this credential belongs to | 
**Name** | **string** |  | 
**Description** | Pointer to **string** |  | [optional] 
**Provider** | **string** |  | 
**Token** | Pointer to **string** | Credential token value; write-only, never returned in GET/LIST responses | [optional] 
**Url** | Pointer to **string** |  | [optional] 
**Email** | Pointer to **string** |  | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 

## Methods

### NewCredential

`func NewCredential(projectId string, name string, provider string, ) *Credential`

NewCredential instantiates a new Credential object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialWithDefaults

`func NewCredentialWithDefaults() *Credential`

NewCredentialWithDefaults instantiates a new Credential object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Credential) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Credential) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Credential) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Credential) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Credential) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Credential) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Credential) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Credential) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Credential) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Credential) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Credential) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Credential) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Credential) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Credential) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Credential) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Credential) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Credential) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Credential) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Credential) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Credential) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetProjectId

`func (o *Credential) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *Credential) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *Credential) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetName

`func (o *Credential) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Credential) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Credential) SetName(v string)`

SetName sets Name field to given value.


### GetDescription

`func (o *Credential) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *Credential) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *Credential) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *Credential) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetProvider

`func (o *Credential) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *Credential) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *Credential) SetProvider(v string)`

SetProvider sets Provider field to given value.


### GetToken

`func (o *Credential) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *Credential) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *Credential) SetToken(v string)`

SetToken sets Token field to given value.

### HasToken

`func (o *Credential) HasToken() bool`

HasToken returns a boolean if a field has been set.

### GetUrl

`func (o *Credential) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *Credential) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *Credential) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *Credential) HasUrl() bool`

HasUrl returns a boolean if a field has been set.

### GetEmail

`func (o *Credential) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *Credential) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *Credential) SetEmail(v string)`

SetEmail sets Email field to given value.

### HasEmail

`func (o *Credential) HasEmail() bool`

HasEmail returns a boolean if a field has been set.

### GetLabels

`func (o *Credential) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *Credential) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *Credential) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *Credential) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *Credential) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *Credential) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *Credential) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *Credential) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


