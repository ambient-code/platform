# WorkflowSkill

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**WorkflowId** | **string** |  | 
**SkillId** | **string** |  | 
**Position** | **int32** |  | 

## Methods

### NewWorkflowSkill

`func NewWorkflowSkill(workflowId string, skillId string, position int32, ) *WorkflowSkill`

NewWorkflowSkill instantiates a new WorkflowSkill object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowSkillWithDefaults

`func NewWorkflowSkillWithDefaults() *WorkflowSkill`

NewWorkflowSkillWithDefaults instantiates a new WorkflowSkill object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *WorkflowSkill) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *WorkflowSkill) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *WorkflowSkill) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *WorkflowSkill) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *WorkflowSkill) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *WorkflowSkill) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *WorkflowSkill) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *WorkflowSkill) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *WorkflowSkill) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *WorkflowSkill) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *WorkflowSkill) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *WorkflowSkill) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *WorkflowSkill) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *WorkflowSkill) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *WorkflowSkill) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *WorkflowSkill) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *WorkflowSkill) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *WorkflowSkill) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *WorkflowSkill) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *WorkflowSkill) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetWorkflowId

`func (o *WorkflowSkill) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *WorkflowSkill) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *WorkflowSkill) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.


### GetSkillId

`func (o *WorkflowSkill) GetSkillId() string`

GetSkillId returns the SkillId field if non-nil, zero value otherwise.

### GetSkillIdOk

`func (o *WorkflowSkill) GetSkillIdOk() (*string, bool)`

GetSkillIdOk returns a tuple with the SkillId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkillId

`func (o *WorkflowSkill) SetSkillId(v string)`

SetSkillId sets SkillId field to given value.


### GetPosition

`func (o *WorkflowSkill) GetPosition() int32`

GetPosition returns the Position field if non-nil, zero value otherwise.

### GetPositionOk

`func (o *WorkflowSkill) GetPositionOk() (*int32, bool)`

GetPositionOk returns a tuple with the Position field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPosition

`func (o *WorkflowSkill) SetPosition(v int32)`

SetPosition sets Position field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


