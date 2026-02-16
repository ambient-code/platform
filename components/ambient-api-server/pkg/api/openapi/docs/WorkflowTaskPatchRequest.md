# WorkflowTaskPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**WorkflowId** | Pointer to **string** |  | [optional] 
**TaskId** | Pointer to **string** |  | [optional] 
**Position** | Pointer to **int32** |  | [optional] 

## Methods

### NewWorkflowTaskPatchRequest

`func NewWorkflowTaskPatchRequest() *WorkflowTaskPatchRequest`

NewWorkflowTaskPatchRequest instantiates a new WorkflowTaskPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowTaskPatchRequestWithDefaults

`func NewWorkflowTaskPatchRequestWithDefaults() *WorkflowTaskPatchRequest`

NewWorkflowTaskPatchRequestWithDefaults instantiates a new WorkflowTaskPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetWorkflowId

`func (o *WorkflowTaskPatchRequest) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *WorkflowTaskPatchRequest) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *WorkflowTaskPatchRequest) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *WorkflowTaskPatchRequest) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetTaskId

`func (o *WorkflowTaskPatchRequest) GetTaskId() string`

GetTaskId returns the TaskId field if non-nil, zero value otherwise.

### GetTaskIdOk

`func (o *WorkflowTaskPatchRequest) GetTaskIdOk() (*string, bool)`

GetTaskIdOk returns a tuple with the TaskId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskId

`func (o *WorkflowTaskPatchRequest) SetTaskId(v string)`

SetTaskId sets TaskId field to given value.

### HasTaskId

`func (o *WorkflowTaskPatchRequest) HasTaskId() bool`

HasTaskId returns a boolean if a field has been set.

### GetPosition

`func (o *WorkflowTaskPatchRequest) GetPosition() int32`

GetPosition returns the Position field if non-nil, zero value otherwise.

### GetPositionOk

`func (o *WorkflowTaskPatchRequest) GetPositionOk() (*int32, bool)`

GetPositionOk returns a tuple with the Position field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPosition

`func (o *WorkflowTaskPatchRequest) SetPosition(v int32)`

SetPosition sets Position field to given value.

### HasPosition

`func (o *WorkflowTaskPatchRequest) HasPosition() bool`

HasPosition returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


