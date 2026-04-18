# AgentPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Prompt** | Pointer to **string** | Update agent prompt (access controlled by RBAC) | [optional] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 

## Methods

### NewAgentPatchRequest

`func NewAgentPatchRequest() *AgentPatchRequest`

NewAgentPatchRequest instantiates a new AgentPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentPatchRequestWithDefaults

`func NewAgentPatchRequestWithDefaults() *AgentPatchRequest`

NewAgentPatchRequestWithDefaults instantiates a new AgentPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *AgentPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AgentPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AgentPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *AgentPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPrompt

`func (o *AgentPatchRequest) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *AgentPatchRequest) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *AgentPatchRequest) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *AgentPatchRequest) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetLabels

`func (o *AgentPatchRequest) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *AgentPatchRequest) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *AgentPatchRequest) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *AgentPatchRequest) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *AgentPatchRequest) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *AgentPatchRequest) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *AgentPatchRequest) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *AgentPatchRequest) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


