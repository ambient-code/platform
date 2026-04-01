# Agent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**ProjectId** | **string** | The project this agent belongs to | 
**Name** | **string** | Human-readable identifier; unique within the project | 
**Prompt** | Pointer to **string** | Defines who this agent is. Mutable via PATCH. Access controlled by RBAC. | [optional] 
**CurrentSessionId** | Pointer to **string** | Denormalized for fast reads — the active session, if any | [optional] [readonly] 
**Labels** | Pointer to **string** |  | [optional] 
**Annotations** | Pointer to **string** |  | [optional] 

## Methods

### NewAgent

`func NewAgent(projectId string, name string, ) *Agent`

NewAgent instantiates a new Agent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentWithDefaults

`func NewAgentWithDefaults() *Agent`

NewAgentWithDefaults instantiates a new Agent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *Agent) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Agent) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Agent) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Agent) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *Agent) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *Agent) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *Agent) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *Agent) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *Agent) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *Agent) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *Agent) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *Agent) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *Agent) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *Agent) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *Agent) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *Agent) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *Agent) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *Agent) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *Agent) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *Agent) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetProjectId

`func (o *Agent) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *Agent) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *Agent) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetName

`func (o *Agent) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Agent) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Agent) SetName(v string)`

SetName sets Name field to given value.


### GetPrompt

`func (o *Agent) GetPrompt() string`

GetPrompt returns the Prompt field if non-nil, zero value otherwise.

### GetPromptOk

`func (o *Agent) GetPromptOk() (*string, bool)`

GetPromptOk returns a tuple with the Prompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrompt

`func (o *Agent) SetPrompt(v string)`

SetPrompt sets Prompt field to given value.

### HasPrompt

`func (o *Agent) HasPrompt() bool`

HasPrompt returns a boolean if a field has been set.

### GetCurrentSessionId

`func (o *Agent) GetCurrentSessionId() string`

GetCurrentSessionId returns the CurrentSessionId field if non-nil, zero value otherwise.

### GetCurrentSessionIdOk

`func (o *Agent) GetCurrentSessionIdOk() (*string, bool)`

GetCurrentSessionIdOk returns a tuple with the CurrentSessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentSessionId

`func (o *Agent) SetCurrentSessionId(v string)`

SetCurrentSessionId sets CurrentSessionId field to given value.

### HasCurrentSessionId

`func (o *Agent) HasCurrentSessionId() bool`

HasCurrentSessionId returns a boolean if a field has been set.

### GetLabels

`func (o *Agent) GetLabels() string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *Agent) GetLabelsOk() (*string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *Agent) SetLabels(v string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *Agent) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetAnnotations

`func (o *Agent) GetAnnotations() string`

GetAnnotations returns the Annotations field if non-nil, zero value otherwise.

### GetAnnotationsOk

`func (o *Agent) GetAnnotationsOk() (*string, bool)`

GetAnnotationsOk returns a tuple with the Annotations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnnotations

`func (o *Agent) SetAnnotations(v string)`

SetAnnotations sets Annotations field to given value.

### HasAnnotations

`func (o *Agent) HasAnnotations() bool`

HasAnnotations returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


