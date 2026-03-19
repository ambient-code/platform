# SessionCheckIn

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**SessionId** | **string** |  | 
**AgentId** | **string** |  | 
**Summary** | Pointer to **string** |  | [optional] 
**Branch** | Pointer to **string** |  | [optional] 
**Worktree** | Pointer to **string** |  | [optional] 
**Pr** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**TestCount** | Pointer to **int32** |  | [optional] 
**NextSteps** | Pointer to **string** |  | [optional] 

## Methods

### NewSessionCheckIn

`func NewSessionCheckIn(sessionId string, agentId string, ) *SessionCheckIn`

NewSessionCheckIn instantiates a new SessionCheckIn object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionCheckInWithDefaults

`func NewSessionCheckInWithDefaults() *SessionCheckIn`

NewSessionCheckInWithDefaults instantiates a new SessionCheckIn object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SessionCheckIn) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SessionCheckIn) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SessionCheckIn) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *SessionCheckIn) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *SessionCheckIn) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *SessionCheckIn) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *SessionCheckIn) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *SessionCheckIn) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *SessionCheckIn) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *SessionCheckIn) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *SessionCheckIn) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *SessionCheckIn) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *SessionCheckIn) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SessionCheckIn) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SessionCheckIn) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *SessionCheckIn) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *SessionCheckIn) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *SessionCheckIn) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *SessionCheckIn) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *SessionCheckIn) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetSessionId

`func (o *SessionCheckIn) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *SessionCheckIn) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *SessionCheckIn) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetAgentId

`func (o *SessionCheckIn) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *SessionCheckIn) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *SessionCheckIn) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.


### GetSummary

`func (o *SessionCheckIn) GetSummary() string`

GetSummary returns the Summary field if non-nil, zero value otherwise.

### GetSummaryOk

`func (o *SessionCheckIn) GetSummaryOk() (*string, bool)`

GetSummaryOk returns a tuple with the Summary field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSummary

`func (o *SessionCheckIn) SetSummary(v string)`

SetSummary sets Summary field to given value.

### HasSummary

`func (o *SessionCheckIn) HasSummary() bool`

HasSummary returns a boolean if a field has been set.

### GetBranch

`func (o *SessionCheckIn) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *SessionCheckIn) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *SessionCheckIn) SetBranch(v string)`

SetBranch sets Branch field to given value.

### HasBranch

`func (o *SessionCheckIn) HasBranch() bool`

HasBranch returns a boolean if a field has been set.

### GetWorktree

`func (o *SessionCheckIn) GetWorktree() string`

GetWorktree returns the Worktree field if non-nil, zero value otherwise.

### GetWorktreeOk

`func (o *SessionCheckIn) GetWorktreeOk() (*string, bool)`

GetWorktreeOk returns a tuple with the Worktree field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorktree

`func (o *SessionCheckIn) SetWorktree(v string)`

SetWorktree sets Worktree field to given value.

### HasWorktree

`func (o *SessionCheckIn) HasWorktree() bool`

HasWorktree returns a boolean if a field has been set.

### GetPr

`func (o *SessionCheckIn) GetPr() string`

GetPr returns the Pr field if non-nil, zero value otherwise.

### GetPrOk

`func (o *SessionCheckIn) GetPrOk() (*string, bool)`

GetPrOk returns a tuple with the Pr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPr

`func (o *SessionCheckIn) SetPr(v string)`

SetPr sets Pr field to given value.

### HasPr

`func (o *SessionCheckIn) HasPr() bool`

HasPr returns a boolean if a field has been set.

### GetPhase

`func (o *SessionCheckIn) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *SessionCheckIn) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *SessionCheckIn) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *SessionCheckIn) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetTestCount

`func (o *SessionCheckIn) GetTestCount() int32`

GetTestCount returns the TestCount field if non-nil, zero value otherwise.

### GetTestCountOk

`func (o *SessionCheckIn) GetTestCountOk() (*int32, bool)`

GetTestCountOk returns a tuple with the TestCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTestCount

`func (o *SessionCheckIn) SetTestCount(v int32)`

SetTestCount sets TestCount field to given value.

### HasTestCount

`func (o *SessionCheckIn) HasTestCount() bool`

HasTestCount returns a boolean if a field has been set.

### GetNextSteps

`func (o *SessionCheckIn) GetNextSteps() string`

GetNextSteps returns the NextSteps field if non-nil, zero value otherwise.

### GetNextStepsOk

`func (o *SessionCheckIn) GetNextStepsOk() (*string, bool)`

GetNextStepsOk returns a tuple with the NextSteps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextSteps

`func (o *SessionCheckIn) SetNextSteps(v string)`

SetNextSteps sets NextSteps field to given value.

### HasNextSteps

`func (o *SessionCheckIn) HasNextSteps() bool`

HasNextSteps returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


