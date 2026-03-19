# IgniteResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Session** | Pointer to [**Session**](Session.md) |  | [optional] 
**IgnitionContext** | Pointer to **string** | Assembled ignition context — Project.prompt + Agent.prompt + Inbox + Session.prompt + peer roster | [optional] 

## Methods

### NewIgniteResponse

`func NewIgniteResponse() *IgniteResponse`

NewIgniteResponse instantiates a new IgniteResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewIgniteResponseWithDefaults

`func NewIgniteResponseWithDefaults() *IgniteResponse`

NewIgniteResponseWithDefaults instantiates a new IgniteResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSession

`func (o *IgniteResponse) GetSession() Session`

GetSession returns the Session field if non-nil, zero value otherwise.

### GetSessionOk

`func (o *IgniteResponse) GetSessionOk() (*Session, bool)`

GetSessionOk returns a tuple with the Session field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSession

`func (o *IgniteResponse) SetSession(v Session)`

SetSession sets Session field to given value.

### HasSession

`func (o *IgniteResponse) HasSession() bool`

HasSession returns a boolean if a field has been set.

### GetIgnitionContext

`func (o *IgniteResponse) GetIgnitionContext() string`

GetIgnitionContext returns the IgnitionContext field if non-nil, zero value otherwise.

### GetIgnitionContextOk

`func (o *IgniteResponse) GetIgnitionContextOk() (*string, bool)`

GetIgnitionContextOk returns a tuple with the IgnitionContext field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIgnitionContext

`func (o *IgniteResponse) SetIgnitionContext(v string)`

SetIgnitionContext sets IgnitionContext field to given value.

### HasIgnitionContext

`func (o *IgniteResponse) HasIgnitionContext() bool`

HasIgnitionContext returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


