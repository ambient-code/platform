# ScheduledSession

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Kind** | Pointer to **string** |  | [optional] 
**Href** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Name** | **string** | Human-readable name for the scheduled session | 
**Description** | Pointer to **string** |  | [optional] 
**ProjectId** | **string** | The project this scheduled session belongs to | 
**AgentId** | Pointer to **string** | Optional agent to run when triggered | [optional] 
**Schedule** | **string** | Cron expression defining the schedule | 
**Timezone** | Pointer to **string** | IANA timezone for the schedule (default UTC) | [optional] 
**Enabled** | Pointer to **bool** | Whether the schedule is active | [optional] 
**SessionPrompt** | Pointer to **string** | Prompt passed to each triggered session | [optional] 
**LastRunAt** | Pointer to **time.Time** |  | [optional] 
**NextRunAt** | Pointer to **time.Time** |  | [optional] 
**Timeout** | Pointer to **int32** | Session timeout in seconds | [optional] 
**InactivityTimeout** | Pointer to **int32** | Session inactivity timeout in seconds | [optional] 
**StopOnRunFinished** | Pointer to **bool** | Whether to stop the session when the run finishes | [optional] 
**RunnerType** | Pointer to **string** | Runner type override for triggered sessions | [optional] 

## Methods

### NewScheduledSession

`func NewScheduledSession(name string, projectId string, schedule string, ) *ScheduledSession`

NewScheduledSession instantiates a new ScheduledSession object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewScheduledSessionWithDefaults

`func NewScheduledSessionWithDefaults() *ScheduledSession`

NewScheduledSessionWithDefaults instantiates a new ScheduledSession object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *ScheduledSession) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *ScheduledSession) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *ScheduledSession) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *ScheduledSession) HasId() bool`

HasId returns a boolean if a field has been set.

### GetKind

`func (o *ScheduledSession) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *ScheduledSession) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *ScheduledSession) SetKind(v string)`

SetKind sets Kind field to given value.

### HasKind

`func (o *ScheduledSession) HasKind() bool`

HasKind returns a boolean if a field has been set.

### GetHref

`func (o *ScheduledSession) GetHref() string`

GetHref returns the Href field if non-nil, zero value otherwise.

### GetHrefOk

`func (o *ScheduledSession) GetHrefOk() (*string, bool)`

GetHrefOk returns a tuple with the Href field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHref

`func (o *ScheduledSession) SetHref(v string)`

SetHref sets Href field to given value.

### HasHref

`func (o *ScheduledSession) HasHref() bool`

HasHref returns a boolean if a field has been set.

### GetCreatedAt

`func (o *ScheduledSession) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *ScheduledSession) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *ScheduledSession) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *ScheduledSession) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *ScheduledSession) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *ScheduledSession) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *ScheduledSession) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *ScheduledSession) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetName

`func (o *ScheduledSession) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ScheduledSession) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ScheduledSession) SetName(v string)`

SetName sets Name field to given value.


### GetDescription

`func (o *ScheduledSession) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ScheduledSession) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ScheduledSession) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ScheduledSession) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetProjectId

`func (o *ScheduledSession) GetProjectId() string`

GetProjectId returns the ProjectId field if non-nil, zero value otherwise.

### GetProjectIdOk

`func (o *ScheduledSession) GetProjectIdOk() (*string, bool)`

GetProjectIdOk returns a tuple with the ProjectId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProjectId

`func (o *ScheduledSession) SetProjectId(v string)`

SetProjectId sets ProjectId field to given value.


### GetAgentId

`func (o *ScheduledSession) GetAgentId() string`

GetAgentId returns the AgentId field if non-nil, zero value otherwise.

### GetAgentIdOk

`func (o *ScheduledSession) GetAgentIdOk() (*string, bool)`

GetAgentIdOk returns a tuple with the AgentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentId

`func (o *ScheduledSession) SetAgentId(v string)`

SetAgentId sets AgentId field to given value.

### HasAgentId

`func (o *ScheduledSession) HasAgentId() bool`

HasAgentId returns a boolean if a field has been set.

### GetSchedule

`func (o *ScheduledSession) GetSchedule() string`

GetSchedule returns the Schedule field if non-nil, zero value otherwise.

### GetScheduleOk

`func (o *ScheduledSession) GetScheduleOk() (*string, bool)`

GetScheduleOk returns a tuple with the Schedule field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedule

`func (o *ScheduledSession) SetSchedule(v string)`

SetSchedule sets Schedule field to given value.


### GetTimezone

`func (o *ScheduledSession) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *ScheduledSession) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *ScheduledSession) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *ScheduledSession) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetEnabled

`func (o *ScheduledSession) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *ScheduledSession) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *ScheduledSession) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.

### HasEnabled

`func (o *ScheduledSession) HasEnabled() bool`

HasEnabled returns a boolean if a field has been set.

### GetSessionPrompt

`func (o *ScheduledSession) GetSessionPrompt() string`

GetSessionPrompt returns the SessionPrompt field if non-nil, zero value otherwise.

### GetSessionPromptOk

`func (o *ScheduledSession) GetSessionPromptOk() (*string, bool)`

GetSessionPromptOk returns a tuple with the SessionPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionPrompt

`func (o *ScheduledSession) SetSessionPrompt(v string)`

SetSessionPrompt sets SessionPrompt field to given value.

### HasSessionPrompt

`func (o *ScheduledSession) HasSessionPrompt() bool`

HasSessionPrompt returns a boolean if a field has been set.

### GetLastRunAt

`func (o *ScheduledSession) GetLastRunAt() time.Time`

GetLastRunAt returns the LastRunAt field if non-nil, zero value otherwise.

### GetLastRunAtOk

`func (o *ScheduledSession) GetLastRunAtOk() (*time.Time, bool)`

GetLastRunAtOk returns a tuple with the LastRunAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastRunAt

`func (o *ScheduledSession) SetLastRunAt(v time.Time)`

SetLastRunAt sets LastRunAt field to given value.

### HasLastRunAt

`func (o *ScheduledSession) HasLastRunAt() bool`

HasLastRunAt returns a boolean if a field has been set.

### GetNextRunAt

`func (o *ScheduledSession) GetNextRunAt() time.Time`

GetNextRunAt returns the NextRunAt field if non-nil, zero value otherwise.

### GetNextRunAtOk

`func (o *ScheduledSession) GetNextRunAtOk() (*time.Time, bool)`

GetNextRunAtOk returns a tuple with the NextRunAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextRunAt

`func (o *ScheduledSession) SetNextRunAt(v time.Time)`

SetNextRunAt sets NextRunAt field to given value.

### HasNextRunAt

`func (o *ScheduledSession) HasNextRunAt() bool`

HasNextRunAt returns a boolean if a field has been set.

### GetTimeout

`func (o *ScheduledSession) GetTimeout() int32`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *ScheduledSession) GetTimeoutOk() (*int32, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *ScheduledSession) SetTimeout(v int32)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *ScheduledSession) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetInactivityTimeout

`func (o *ScheduledSession) GetInactivityTimeout() int32`

GetInactivityTimeout returns the InactivityTimeout field if non-nil, zero value otherwise.

### GetInactivityTimeoutOk

`func (o *ScheduledSession) GetInactivityTimeoutOk() (*int32, bool)`

GetInactivityTimeoutOk returns a tuple with the InactivityTimeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInactivityTimeout

`func (o *ScheduledSession) SetInactivityTimeout(v int32)`

SetInactivityTimeout sets InactivityTimeout field to given value.

### HasInactivityTimeout

`func (o *ScheduledSession) HasInactivityTimeout() bool`

HasInactivityTimeout returns a boolean if a field has been set.

### GetStopOnRunFinished

`func (o *ScheduledSession) GetStopOnRunFinished() bool`

GetStopOnRunFinished returns the StopOnRunFinished field if non-nil, zero value otherwise.

### GetStopOnRunFinishedOk

`func (o *ScheduledSession) GetStopOnRunFinishedOk() (*bool, bool)`

GetStopOnRunFinishedOk returns a tuple with the StopOnRunFinished field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStopOnRunFinished

`func (o *ScheduledSession) SetStopOnRunFinished(v bool)`

SetStopOnRunFinished sets StopOnRunFinished field to given value.

### HasStopOnRunFinished

`func (o *ScheduledSession) HasStopOnRunFinished() bool`

HasStopOnRunFinished returns a boolean if a field has been set.

### GetRunnerType

`func (o *ScheduledSession) GetRunnerType() string`

GetRunnerType returns the RunnerType field if non-nil, zero value otherwise.

### GetRunnerTypeOk

`func (o *ScheduledSession) GetRunnerTypeOk() (*string, bool)`

GetRunnerTypeOk returns a tuple with the RunnerType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunnerType

`func (o *ScheduledSession) SetRunnerType(v string)`

SetRunnerType sets RunnerType field to given value.

### HasRunnerType

`func (o *ScheduledSession) HasRunnerType() bool`

HasRunnerType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


