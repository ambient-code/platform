# SessionCheckInPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SessionId** | Pointer to **string** |  | [optional] 
**AgentId** | Pointer to **string** |  | [optional] 
**Summary** | Pointer to **string** |  | [optional] 
**Branch** | Pointer to **string** |  | [optional] 
**Worktree** | Pointer to **string** |  | [optional] 
**Pr** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**TestCount** | Pointer to **int32** |  | [optional] 
**NextSteps** | Pointer to **string** |  | [optional] 

## Methods

### NewSessionCheckInPatchRequest

`func NewSessionCheckInPatchRequest() *SessionCheckInPatchRequest`

NewSessionCheckInPatchRequest instantiates a new SessionCheckInPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionCheckInPatchRequestWithDefaults

`func NewSessionCheckInPatchRequestWithDefaults() *SessionCheckInPatchRequest`

NewSessionCheckInPatchRequestWithDefaults instantiates a new SessionCheckInPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSessionId

`func (o *SessionCheckInPatchRequest) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *SessionCheckInPatchRequest) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *SessionCheckInPatchRequest) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.

### HasSessionId

`func (o *SessionCheckInPatchRequest) HasSessionId() bool`

HasSessionId returns a boolean if a field has been set.

### GetAgentId

`func (o *SessionCheckInPatchRequest) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *SessionCheckInPatchRequest) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *SessionCheckInPatchRequest) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.

### HasAgentId

`func (o *SessionCheckInPatchRequest) HasAgentId() bool`

HasAgentId returns a boolean if a field has been set.

### GetSummary

`func (o *SessionCheckInPatchRequest) GetSummary() string`

GetSummary returns the Summary field if non-nil, zero value otherwise.

### GetSummaryOk

`func (o *SessionCheckInPatchRequest) GetSummaryOk() (*string, bool)`

GetSummaryOk returns a tuple with the Summary field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSummary

`func (o *SessionCheckInPatchRequest) SetSummary(v string)`

SetSummary sets Summary field to given value.

### HasSummary

`func (o *SessionCheckInPatchRequest) HasSummary() bool`

HasSummary returns a boolean if a field has been set.

### GetBranch

`func (o *SessionCheckInPatchRequest) GetBranch() string`

GetBranch returns the Branch field if non-nil, zero value otherwise.

### GetBranchOk

`func (o *SessionCheckInPatchRequest) GetBranchOk() (*string, bool)`

GetBranchOk returns a tuple with the Branch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBranch

`func (o *SessionCheckInPatchRequest) SetBranch(v string)`

SetBranch sets Branch field to given value.

### HasBranch

`func (o *SessionCheckInPatchRequest) HasBranch() bool`

HasBranch returns a boolean if a field has been set.

### GetWorktree

`func (o *SessionCheckInPatchRequest) GetWorktree() string`

GetWorktree returns the Worktree field if non-nil, zero value otherwise.

### GetWorktreeOk

`func (o *SessionCheckInPatchRequest) GetWorktreeOk() (*string, bool)`

GetWorktreeOk returns a tuple with the Worktree field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorktree

`func (o *SessionCheckInPatchRequest) SetWorktree(v string)`

SetWorktree sets Worktree field to given value.

### HasWorktree

`func (o *SessionCheckInPatchRequest) HasWorktree() bool`

HasWorktree returns a boolean if a field has been set.

### GetPr

`func (o *SessionCheckInPatchRequest) GetPr() string`

GetPr returns the Pr field if non-nil, zero value otherwise.

### GetPrOk

`func (o *SessionCheckInPatchRequest) GetPrOk() (*string, bool)`

GetPrOk returns a tuple with the Pr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPr

`func (o *SessionCheckInPatchRequest) SetPr(v string)`

SetPr sets Pr field to given value.

### HasPr

`func (o *SessionCheckInPatchRequest) HasPr() bool`

HasPr returns a boolean if a field has been set.

### GetPhase

`func (o *SessionCheckInPatchRequest) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *SessionCheckInPatchRequest) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *SessionCheckInPatchRequest) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *SessionCheckInPatchRequest) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetTestCount

`func (o *SessionCheckInPatchRequest) GetTestCount() int32`

GetTestCount returns the TestCount field if non-nil, zero value otherwise.

### GetTestCountOk

`func (o *SessionCheckInPatchRequest) GetTestCountOk() (*int32, bool)`

GetTestCountOk returns a tuple with the TestCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTestCount

`func (o *SessionCheckInPatchRequest) SetTestCount(v int32)`

SetTestCount sets TestCount field to given value.

### HasTestCount

`func (o *SessionCheckInPatchRequest) HasTestCount() bool`

HasTestCount returns a boolean if a field has been set.

### GetNextSteps

`func (o *SessionCheckInPatchRequest) GetNextSteps() string`

GetNextSteps returns the NextSteps field if non-nil, zero value otherwise.

### GetNextStepsOk

`func (o *SessionCheckInPatchRequest) GetNextStepsOk() (*string, bool)`

GetNextStepsOk returns a tuple with the NextSteps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextSteps

`func (o *SessionCheckInPatchRequest) SetNextSteps(v string)`

SetNextSteps sets NextSteps field to given value.

### HasNextSteps

`func (o *SessionCheckInPatchRequest) HasNextSteps() bool`

HasNextSteps returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


