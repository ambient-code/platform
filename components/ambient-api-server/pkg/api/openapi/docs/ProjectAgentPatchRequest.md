# ProjectAgentPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AgentVersion** | Pointer to **int32** | Re-pin this ProjectAgent to a different Agent version | [optional] 

## Methods

### NewProjectAgentPatchRequest

`func NewProjectAgentPatchRequest() *ProjectAgentPatchRequest`

NewProjectAgentPatchRequest instantiates a new ProjectAgentPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewProjectAgentPatchRequestWithDefaults

`func NewProjectAgentPatchRequestWithDefaults() *ProjectAgentPatchRequest`

NewProjectAgentPatchRequestWithDefaults instantiates a new ProjectAgentPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgentVersion

`func (o *ProjectAgentPatchRequest) GetAgentVersion() int32`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *ProjectAgentPatchRequest) GetAgentVersionOk() (*int32, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *ProjectAgentPatchRequest) SetAgentVersion(v int32)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *ProjectAgentPatchRequest) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


