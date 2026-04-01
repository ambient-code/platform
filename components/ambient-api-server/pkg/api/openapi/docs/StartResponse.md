# StartResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Session** | Pointer to [**Session**](Session.md) |  | [optional] 
**IgnitionPrompt** | Pointer to **string** | Assembled start prompt — Agent.prompt + Inbox + Session.prompt + peer roster | [optional] 

## Methods

### NewStartResponse

`func NewStartResponse() *StartResponse`

NewStartResponse instantiates a new StartResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStartResponseWithDefaults

`func NewStartResponseWithDefaults() *StartResponse`

NewStartResponseWithDefaults instantiates a new StartResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSession

`func (o *StartResponse) GetSession() Session`

GetSession returns the Session field if non-nil, zero value otherwise.

### GetSessionOk

`func (o *StartResponse) GetSessionOk() (*Session, bool)`

GetSessionOk returns a tuple with the Session field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSession

`func (o *StartResponse) SetSession(v Session)`

SetSession sets Session field to given value.

### HasSession

`func (o *StartResponse) HasSession() bool`

HasSession returns a boolean if a field has been set.

### GetIgnitionPrompt

`func (o *StartResponse) GetIgnitionPrompt() string`

GetIgnitionPrompt returns the IgnitionPrompt field if non-nil, zero value otherwise.

### GetIgnitionPromptOk

`func (o *StartResponse) GetIgnitionPromptOk() (*string, bool)`

GetIgnitionPromptOk returns a tuple with the IgnitionPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIgnitionPrompt

`func (o *StartResponse) SetIgnitionPrompt(v string)`

SetIgnitionPrompt sets IgnitionPrompt field to given value.

### HasIgnitionPrompt

`func (o *StartResponse) HasIgnitionPrompt() bool`

HasIgnitionPrompt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


