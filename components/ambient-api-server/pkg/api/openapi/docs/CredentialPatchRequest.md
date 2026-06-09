# CredentialPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Provider** | Pointer to **string** |  | [optional] 
**Token** | Pointer to **string** | Credential token value; write-only, never returned in GET/LIST responses | [optional] 
**Url** | Pointer to **string** |  | [optional] 
**Email** | Pointer to **string** |  | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 

## Methods

### NewCredentialPatchRequest

`func NewCredentialPatchRequest() *CredentialPatchRequest`

NewCredentialPatchRequest instantiates a new CredentialPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialPatchRequestWithDefaults

`func NewCredentialPatchRequestWithDefaults() *CredentialPatchRequest`

NewCredentialPatchRequestWithDefaults instantiates a new CredentialPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *CredentialPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CredentialPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CredentialPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *CredentialPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDescription

`func (o *CredentialPatchRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CredentialPatchRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CredentialPatchRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CredentialPatchRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetProvider

`func (o *CredentialPatchRequest) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *CredentialPatchRequest) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *CredentialPatchRequest) SetProvider(v string)`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *CredentialPatchRequest) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### GetToken

`func (o *CredentialPatchRequest) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *CredentialPatchRequest) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *CredentialPatchRequest) SetToken(v string)`

SetToken sets Token field to given value.

### HasToken

`func (o *CredentialPatchRequest) HasToken() bool`

HasToken returns a boolean if a field has been set.

### GetUrl

`func (o *CredentialPatchRequest) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *CredentialPatchRequest) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *CredentialPatchRequest) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *CredentialPatchRequest) HasUrl() bool`

HasUrl returns a boolean if a field has been set.

### GetEmail

`func (o *CredentialPatchRequest) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *CredentialPatchRequest) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *CredentialPatchRequest) SetEmail(v string)`

SetEmail sets Email field to given value.

### HasEmail

`func (o *CredentialPatchRequest) HasEmail() bool`

HasEmail returns a boolean if a field has been set.

### GetLabels

`func (o *CredentialPatchRequest) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *CredentialPatchRequest) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *CredentialPatchRequest) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *CredentialPatchRequest) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *CredentialPatchRequest) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *CredentialPatchRequest) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *CredentialPatchRequest) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *CredentialPatchRequest) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


