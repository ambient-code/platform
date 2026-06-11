# ScheduledSessionPatchRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**AgentId** | Pointer to **string** |  | [optional] 
**Schedule** | Pointer to **string** |  | [optional] 
**Timezone** | Pointer to **string** |  | [optional] 
**Enabled** | Pointer to **bool** |  | [optional] 
**SessionPrompt** | Pointer to **string** |  | [optional] 
**Timeout** | Pointer to **int32** |  | [optional] 
**InactivityTimeout** | Pointer to **int32** |  | [optional] 
**StopOnRunFinished** | Pointer to **bool** |  | [optional] 
**RunnerType** | Pointer to **string** |  | [optional] 

## Methods

### NewScheduledSessionPatchRequest

`func NewScheduledSessionPatchRequest() *ScheduledSessionPatchRequest`

NewScheduledSessionPatchRequest instantiates a new ScheduledSessionPatchRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewScheduledSessionPatchRequestWithDefaults

`func NewScheduledSessionPatchRequestWithDefaults() *ScheduledSessionPatchRequest`

NewScheduledSessionPatchRequestWithDefaults instantiates a new ScheduledSessionPatchRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ScheduledSessionPatchRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ScheduledSessionPatchRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ScheduledSessionPatchRequest) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ScheduledSessionPatchRequest) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDescription

`func (o *ScheduledSessionPatchRequest) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ScheduledSessionPatchRequest) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ScheduledSessionPatchRequest) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ScheduledSessionPatchRequest) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetAgentId

`func (o *ScheduledSessionPatchRequest) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *ScheduledSessionPatchRequest) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *ScheduledSessionPatchRequest) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.

### HasAgentId

`func (o *ScheduledSessionPatchRequest) HasAgentId() bool`

HasAgentId returns a boolean if a field has been set.

### GetSchedule

`func (o *ScheduledSessionPatchRequest) GetSchedule() string`

GetSchedule returns the Schedule field if non-nil, zero value otherwise.

### GetScheduleOk

`func (o *ScheduledSessionPatchRequest) GetScheduleOk() (*string, bool)`

GetScheduleOk returns a tuple with the Schedule field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedule

`func (o *ScheduledSessionPatchRequest) SetSchedule(v string)`

SetSchedule sets Schedule field to given value.

### HasSchedule

`func (o *ScheduledSessionPatchRequest) HasSchedule() bool`

HasSchedule returns a boolean if a field has been set.

### GetTimezone

`func (o *ScheduledSessionPatchRequest) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *ScheduledSessionPatchRequest) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *ScheduledSessionPatchRequest) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *ScheduledSessionPatchRequest) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetEnabled

`func (o *ScheduledSessionPatchRequest) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *ScheduledSessionPatchRequest) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *ScheduledSessionPatchRequest) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.

### HasEnabled

`func (o *ScheduledSessionPatchRequest) HasEnabled() bool`

HasEnabled returns a boolean if a field has been set.

### GetSessionPrompt

`func (o *ScheduledSessionPatchRequest) GetSessionPrompt() string`

GetSessionPrompt returns the SessionPrompt field if non-nil, zero value otherwise.

### GetSessionPromptOk

`func (o *ScheduledSessionPatchRequest) GetSessionPromptOk() (*string, bool)`

GetSessionPromptOk returns a tuple with the SessionPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionPrompt

`func (o *ScheduledSessionPatchRequest) SetSessionPrompt(v string)`

SetSessionPrompt sets SessionPrompt field to given value.

### HasSessionPrompt

`func (o *ScheduledSessionPatchRequest) HasSessionPrompt() bool`

HasSessionPrompt returns a boolean if a field has been set.

### GetTimeout

`func (o *ScheduledSessionPatchRequest) GetTimeout() int32`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *ScheduledSessionPatchRequest) GetTimeoutOk() (*int32, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *ScheduledSessionPatchRequest) SetTimeout(v int32)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *ScheduledSessionPatchRequest) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetInactivityTimeout

`func (o *ScheduledSessionPatchRequest) GetInactivityTimeout() int32`

GetInactivityTimeout returns the InactivityTimeout field if non-nil, zero value otherwise.

### GetInactivityTimeoutOk

`func (o *ScheduledSessionPatchRequest) GetInactivityTimeoutOk() (*int32, bool)`

GetInactivityTimeoutOk returns a tuple with the InactivityTimeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInactivityTimeout

`func (o *ScheduledSessionPatchRequest) SetInactivityTimeout(v int32)`

SetInactivityTimeout sets InactivityTimeout field to given value.

### HasInactivityTimeout

`func (o *ScheduledSessionPatchRequest) HasInactivityTimeout() bool`

HasInactivityTimeout returns a boolean if a field has been set.

### GetStopOnRunFinished

`func (o *ScheduledSessionPatchRequest) GetStopOnRunFinished() bool`

GetStopOnRunFinished returns the StopOnRunFinished field if non-nil, zero value otherwise.

### GetStopOnRunFinishedOk

`func (o *ScheduledSessionPatchRequest) GetStopOnRunFinishedOk() (*bool, bool)`

GetStopOnRunFinishedOk returns a tuple with the StopOnRunFinished field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStopOnRunFinished

`func (o *ScheduledSessionPatchRequest) SetStopOnRunFinished(v bool)`

SetStopOnRunFinished sets StopOnRunFinished field to given value.

### HasStopOnRunFinished

`func (o *ScheduledSessionPatchRequest) HasStopOnRunFinished() bool`

HasStopOnRunFinished returns a boolean if a field has been set.

### GetRunnerType

`func (o *ScheduledSessionPatchRequest) GetRunnerType() string`

GetRunnerType returns the RunnerType field if non-nil, zero value otherwise.

### GetRunnerTypeOk

`func (o *ScheduledSessionPatchRequest) GetRunnerTypeOk() (*string, bool)`

GetRunnerTypeOk returns a tuple with the RunnerType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunnerType

`func (o *ScheduledSessionPatchRequest) SetRunnerType(v string)`

SetRunnerType sets RunnerType field to given value.

### HasRunnerType

`func (o *ScheduledSessionPatchRequest) HasRunnerType() bool`

HasRunnerType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


