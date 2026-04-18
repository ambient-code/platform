# StartRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Prompt** | Pointer to **string** | Task scope for this specific run (Session.prompt) | [optional] 

## Methods

### NewStartRequest

`func NewStartRequest() *StartRequest`

NewStartRequest instantiates a new StartRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStartRequestWithDefaults

`func NewStartRequestWithDefaults() *StartRequest`

NewStartRequestWithDefaults instantiates a new StartRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetPrompt

`func (o *StartRequest) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *StartRequest) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *StartRequest) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *StartRequest) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


