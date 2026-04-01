# CredentialTokenResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CredentialId** | **string** | ID of the credential | 
**Provider** | **string** | Provider type for this credential | 
**Token** | **string** | Decrypted token value | 

## Methods

### NewCredentialTokenResponse

`func NewCredentialTokenResponse(credentialId string, provider string, token string, ) *CredentialTokenResponse`

NewCredentialTokenResponse instantiates a new CredentialTokenResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialTokenResponseWithDefaults

`func NewCredentialTokenResponseWithDefaults() *CredentialTokenResponse`

NewCredentialTokenResponseWithDefaults instantiates a new CredentialTokenResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCredentialId

`func (o *CredentialTokenResponse) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *CredentialTokenResponse) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *CredentialTokenResponse) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.


### GetProvider

`func (o *CredentialTokenResponse) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *CredentialTokenResponse) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *CredentialTokenResponse) SetProvider(v string)`

SetProvider sets Provider field to given value.


### GetToken

`func (o *CredentialTokenResponse) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *CredentialTokenResponse) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *CredentialTokenResponse) SetToken(v string)`

SetToken sets Token field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


