# \DefaultAPI

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiAmbientV1AgentsGet**](DefaultAPI.md#ApiAmbientV1AgentsGet) | **Get** /api/ambient/v1/agents | Returns a list of agents
[**ApiAmbientV1AgentsIdGet**](DefaultAPI.md#ApiAmbientV1AgentsIdGet) | **Get** /api/ambient/v1/agents/{id} | Get an agent by id
[**ApiAmbientV1AgentsIdPatch**](DefaultAPI.md#ApiAmbientV1AgentsIdPatch) | **Patch** /api/ambient/v1/agents/{id} | Update an agent
[**ApiAmbientV1AgentsPost**](DefaultAPI.md#ApiAmbientV1AgentsPost) | **Post** /api/ambient/v1/agents | Create a new agent
[**ApiAmbientV1ProjectSettingsGet**](DefaultAPI.md#ApiAmbientV1ProjectSettingsGet) | **Get** /api/ambient/v1/project_settings | Returns a list of project settings
[**ApiAmbientV1ProjectSettingsIdDelete**](DefaultAPI.md#ApiAmbientV1ProjectSettingsIdDelete) | **Delete** /api/ambient/v1/project_settings/{id} | Delete a project settings by id
[**ApiAmbientV1ProjectSettingsIdGet**](DefaultAPI.md#ApiAmbientV1ProjectSettingsIdGet) | **Get** /api/ambient/v1/project_settings/{id} | Get a project settings by id
[**ApiAmbientV1ProjectSettingsIdPatch**](DefaultAPI.md#ApiAmbientV1ProjectSettingsIdPatch) | **Patch** /api/ambient/v1/project_settings/{id} | Update a project settings
[**ApiAmbientV1ProjectSettingsPost**](DefaultAPI.md#ApiAmbientV1ProjectSettingsPost) | **Post** /api/ambient/v1/project_settings | Create a new project settings
[**ApiAmbientV1ProjectsGet**](DefaultAPI.md#ApiAmbientV1ProjectsGet) | **Get** /api/ambient/v1/projects | Returns a list of projects
[**ApiAmbientV1ProjectsIdDelete**](DefaultAPI.md#ApiAmbientV1ProjectsIdDelete) | **Delete** /api/ambient/v1/projects/{id} | Delete a project by id
[**ApiAmbientV1ProjectsIdGet**](DefaultAPI.md#ApiAmbientV1ProjectsIdGet) | **Get** /api/ambient/v1/projects/{id} | Get a project by id
[**ApiAmbientV1ProjectsIdPatch**](DefaultAPI.md#ApiAmbientV1ProjectsIdPatch) | **Patch** /api/ambient/v1/projects/{id} | Update a project
[**ApiAmbientV1ProjectsPost**](DefaultAPI.md#ApiAmbientV1ProjectsPost) | **Post** /api/ambient/v1/projects | Create a new project
[**ApiAmbientV1RoleBindingsGet**](DefaultAPI.md#ApiAmbientV1RoleBindingsGet) | **Get** /api/ambient/v1/role_bindings | Returns a list of roleBindings
[**ApiAmbientV1RoleBindingsIdGet**](DefaultAPI.md#ApiAmbientV1RoleBindingsIdGet) | **Get** /api/ambient/v1/role_bindings/{id} | Get an roleBinding by id
[**ApiAmbientV1RoleBindingsIdPatch**](DefaultAPI.md#ApiAmbientV1RoleBindingsIdPatch) | **Patch** /api/ambient/v1/role_bindings/{id} | Update an roleBinding
[**ApiAmbientV1RoleBindingsPost**](DefaultAPI.md#ApiAmbientV1RoleBindingsPost) | **Post** /api/ambient/v1/role_bindings | Create a new roleBinding
[**ApiAmbientV1RolesGet**](DefaultAPI.md#ApiAmbientV1RolesGet) | **Get** /api/ambient/v1/roles | Returns a list of roles
[**ApiAmbientV1RolesIdGet**](DefaultAPI.md#ApiAmbientV1RolesIdGet) | **Get** /api/ambient/v1/roles/{id} | Get an role by id
[**ApiAmbientV1RolesIdPatch**](DefaultAPI.md#ApiAmbientV1RolesIdPatch) | **Patch** /api/ambient/v1/roles/{id} | Update an role
[**ApiAmbientV1RolesPost**](DefaultAPI.md#ApiAmbientV1RolesPost) | **Post** /api/ambient/v1/roles | Create a new role
[**ApiAmbientV1SessionsGet**](DefaultAPI.md#ApiAmbientV1SessionsGet) | **Get** /api/ambient/v1/sessions | Returns a list of sessions
[**ApiAmbientV1SessionsIdDelete**](DefaultAPI.md#ApiAmbientV1SessionsIdDelete) | **Delete** /api/ambient/v1/sessions/{id} | Delete a session by id
[**ApiAmbientV1SessionsIdGet**](DefaultAPI.md#ApiAmbientV1SessionsIdGet) | **Get** /api/ambient/v1/sessions/{id} | Get an session by id
[**ApiAmbientV1SessionsIdMessagesGet**](DefaultAPI.md#ApiAmbientV1SessionsIdMessagesGet) | **Get** /api/ambient/v1/sessions/{id}/messages | List or stream session messages
[**ApiAmbientV1SessionsIdMessagesPost**](DefaultAPI.md#ApiAmbientV1SessionsIdMessagesPost) | **Post** /api/ambient/v1/sessions/{id}/messages | Push a message to a session
[**ApiAmbientV1SessionsIdPatch**](DefaultAPI.md#ApiAmbientV1SessionsIdPatch) | **Patch** /api/ambient/v1/sessions/{id} | Update an session
[**ApiAmbientV1SessionsIdStartPost**](DefaultAPI.md#ApiAmbientV1SessionsIdStartPost) | **Post** /api/ambient/v1/sessions/{id}/start | Start a session
[**ApiAmbientV1SessionsIdStatusPatch**](DefaultAPI.md#ApiAmbientV1SessionsIdStatusPatch) | **Patch** /api/ambient/v1/sessions/{id}/status | Update session status fields
[**ApiAmbientV1SessionsIdStopPost**](DefaultAPI.md#ApiAmbientV1SessionsIdStopPost) | **Post** /api/ambient/v1/sessions/{id}/stop | Stop a session
[**ApiAmbientV1SessionsPost**](DefaultAPI.md#ApiAmbientV1SessionsPost) | **Post** /api/ambient/v1/sessions | Create a new session
[**ApiAmbientV1UsersGet**](DefaultAPI.md#ApiAmbientV1UsersGet) | **Get** /api/ambient/v1/users | Returns a list of users
[**ApiAmbientV1UsersIdGet**](DefaultAPI.md#ApiAmbientV1UsersIdGet) | **Get** /api/ambient/v1/users/{id} | Get an user by id
[**ApiAmbientV1UsersIdPatch**](DefaultAPI.md#ApiAmbientV1UsersIdPatch) | **Patch** /api/ambient/v1/users/{id} | Update an user
[**ApiAmbientV1UsersPost**](DefaultAPI.md#ApiAmbientV1UsersPost) | **Post** /api/ambient/v1/users | Create a new user



## ApiAmbientV1AgentsGet

> AgentList ApiAmbientV1AgentsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of agents

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1AgentsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1AgentsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1AgentsGet`: AgentList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1AgentsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1AgentsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**AgentList**](AgentList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1AgentsIdGet

> Agent ApiAmbientV1AgentsIdGet(ctx, id).Execute()

Get an agent by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1AgentsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1AgentsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1AgentsIdGet`: Agent
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1AgentsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1AgentsIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Agent**](Agent.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1AgentsIdPatch

> Agent ApiAmbientV1AgentsIdPatch(ctx, id).AgentPatchRequest(agentPatchRequest).Execute()

Update an agent

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	agentPatchRequest := *openapiclient.NewAgentPatchRequest() // AgentPatchRequest | Updated agent data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1AgentsIdPatch(context.Background(), id).AgentPatchRequest(agentPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1AgentsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1AgentsIdPatch`: Agent
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1AgentsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1AgentsIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **agentPatchRequest** | [**AgentPatchRequest**](AgentPatchRequest.md) | Updated agent data | 

### Return type

[**Agent**](Agent.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1AgentsPost

> Agent ApiAmbientV1AgentsPost(ctx).Agent(agent).Execute()

Create a new agent

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	agent := *openapiclient.NewAgent("ProjectId_example", "OwnerUserId_example", "Name_example") // Agent | Agent data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1AgentsPost(context.Background()).Agent(agent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1AgentsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1AgentsPost`: Agent
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1AgentsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1AgentsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **agent** | [**Agent**](Agent.md) | Agent data | 

### Return type

[**Agent**](Agent.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectSettingsGet

> ProjectSettingsList ApiAmbientV1ProjectSettingsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of project settings

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectSettingsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectSettingsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectSettingsGet`: ProjectSettingsList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectSettingsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectSettingsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**ProjectSettingsList**](ProjectSettingsList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectSettingsIdDelete

> ApiAmbientV1ProjectSettingsIdDelete(ctx, id).Execute()

Delete a project settings by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectSettingsIdDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectSettingsIdDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectSettingsIdDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectSettingsIdGet

> ProjectSettings ApiAmbientV1ProjectSettingsIdGet(ctx, id).Execute()

Get a project settings by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectSettingsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectSettingsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectSettingsIdGet`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectSettingsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectSettingsIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ProjectSettings**](ProjectSettings.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectSettingsIdPatch

> ProjectSettings ApiAmbientV1ProjectSettingsIdPatch(ctx, id).ProjectSettingsPatchRequest(projectSettingsPatchRequest).Execute()

Update a project settings

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	projectSettingsPatchRequest := *openapiclient.NewProjectSettingsPatchRequest() // ProjectSettingsPatchRequest | Updated project settings data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectSettingsIdPatch(context.Background(), id).ProjectSettingsPatchRequest(projectSettingsPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectSettingsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectSettingsIdPatch`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectSettingsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectSettingsIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **projectSettingsPatchRequest** | [**ProjectSettingsPatchRequest**](ProjectSettingsPatchRequest.md) | Updated project settings data | 

### Return type

[**ProjectSettings**](ProjectSettings.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectSettingsPost

> ProjectSettings ApiAmbientV1ProjectSettingsPost(ctx).ProjectSettings(projectSettings).Execute()

Create a new project settings

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	projectSettings := *openapiclient.NewProjectSettings("ProjectId_example") // ProjectSettings | Project settings data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectSettingsPost(context.Background()).ProjectSettings(projectSettings).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectSettingsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectSettingsPost`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectSettingsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectSettingsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **projectSettings** | [**ProjectSettings**](ProjectSettings.md) | Project settings data | 

### Return type

[**ProjectSettings**](ProjectSettings.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectsGet

> ProjectList ApiAmbientV1ProjectsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of projects

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectsGet`: ProjectList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**ProjectList**](ProjectList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectsIdDelete

> ApiAmbientV1ProjectsIdDelete(ctx, id).Execute()

Delete a project by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectsIdDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectsIdDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectsIdDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectsIdGet

> Project ApiAmbientV1ProjectsIdGet(ctx, id).Execute()

Get a project by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectsIdGet`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectsIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Project**](Project.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectsIdPatch

> Project ApiAmbientV1ProjectsIdPatch(ctx, id).ProjectPatchRequest(projectPatchRequest).Execute()

Update a project

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	projectPatchRequest := *openapiclient.NewProjectPatchRequest() // ProjectPatchRequest | Updated project data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectsIdPatch(context.Background(), id).ProjectPatchRequest(projectPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectsIdPatch`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectsIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **projectPatchRequest** | [**ProjectPatchRequest**](ProjectPatchRequest.md) | Updated project data | 

### Return type

[**Project**](Project.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1ProjectsPost

> Project ApiAmbientV1ProjectsPost(ctx).Project(project).Execute()

Create a new project

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	project := *openapiclient.NewProject("Name_example") // Project | Project data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1ProjectsPost(context.Background()).Project(project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1ProjectsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1ProjectsPost`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1ProjectsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1ProjectsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **project** | [**Project**](Project.md) | Project data | 

### Return type

[**Project**](Project.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RoleBindingsGet

> RoleBindingList ApiAmbientV1RoleBindingsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of roleBindings

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RoleBindingsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RoleBindingsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RoleBindingsGet`: RoleBindingList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RoleBindingsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RoleBindingsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**RoleBindingList**](RoleBindingList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RoleBindingsIdGet

> RoleBinding ApiAmbientV1RoleBindingsIdGet(ctx, id).Execute()

Get an roleBinding by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RoleBindingsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RoleBindingsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RoleBindingsIdGet`: RoleBinding
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RoleBindingsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RoleBindingsIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RoleBinding**](RoleBinding.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RoleBindingsIdPatch

> RoleBinding ApiAmbientV1RoleBindingsIdPatch(ctx, id).RoleBindingPatchRequest(roleBindingPatchRequest).Execute()

Update an roleBinding

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	roleBindingPatchRequest := *openapiclient.NewRoleBindingPatchRequest() // RoleBindingPatchRequest | Updated roleBinding data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RoleBindingsIdPatch(context.Background(), id).RoleBindingPatchRequest(roleBindingPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RoleBindingsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RoleBindingsIdPatch`: RoleBinding
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RoleBindingsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RoleBindingsIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **roleBindingPatchRequest** | [**RoleBindingPatchRequest**](RoleBindingPatchRequest.md) | Updated roleBinding data | 

### Return type

[**RoleBinding**](RoleBinding.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RoleBindingsPost

> RoleBinding ApiAmbientV1RoleBindingsPost(ctx).RoleBinding(roleBinding).Execute()

Create a new roleBinding

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	roleBinding := *openapiclient.NewRoleBinding("UserId_example", "RoleId_example", "Scope_example") // RoleBinding | RoleBinding data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RoleBindingsPost(context.Background()).RoleBinding(roleBinding).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RoleBindingsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RoleBindingsPost`: RoleBinding
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RoleBindingsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RoleBindingsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **roleBinding** | [**RoleBinding**](RoleBinding.md) | RoleBinding data | 

### Return type

[**RoleBinding**](RoleBinding.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RolesGet

> RoleList ApiAmbientV1RolesGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of roles

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RolesGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RolesGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RolesGet`: RoleList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RolesGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RolesGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**RoleList**](RoleList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RolesIdGet

> Role ApiAmbientV1RolesIdGet(ctx, id).Execute()

Get an role by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RolesIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RolesIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RolesIdGet`: Role
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RolesIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RolesIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Role**](Role.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RolesIdPatch

> Role ApiAmbientV1RolesIdPatch(ctx, id).RolePatchRequest(rolePatchRequest).Execute()

Update an role

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	rolePatchRequest := *openapiclient.NewRolePatchRequest() // RolePatchRequest | Updated role data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RolesIdPatch(context.Background(), id).RolePatchRequest(rolePatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RolesIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RolesIdPatch`: Role
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RolesIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RolesIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **rolePatchRequest** | [**RolePatchRequest**](RolePatchRequest.md) | Updated role data | 

### Return type

[**Role**](Role.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1RolesPost

> Role ApiAmbientV1RolesPost(ctx).Role(role).Execute()

Create a new role

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	role := *openapiclient.NewRole("Name_example") // Role | Role data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1RolesPost(context.Background()).Role(role).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1RolesPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1RolesPost`: Role
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1RolesPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1RolesPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **role** | [**Role**](Role.md) | Role data | 

### Return type

[**Role**](Role.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsGet

> SessionList ApiAmbientV1SessionsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of sessions

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsGet`: SessionList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**SessionList**](SessionList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdDelete

> ApiAmbientV1SessionsIdDelete(ctx, id).Execute()

Delete a session by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdGet

> Session ApiAmbientV1SessionsIdGet(ctx, id).Execute()

Get an session by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdGet`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdMessagesGet

> []SessionMessage ApiAmbientV1SessionsIdMessagesGet(ctx, id).AfterSeq(afterSeq).Execute()

List or stream session messages



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	afterSeq := int64(789) // int64 | Return only messages with seq greater than this value (default 0 = all messages) (optional) (default to 0)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdMessagesGet(context.Background(), id).AfterSeq(afterSeq).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdMessagesGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdMessagesGet`: []SessionMessage
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdMessagesGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdMessagesGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **afterSeq** | **int64** | Return only messages with seq greater than this value (default 0 &#x3D; all messages) | [default to 0]

### Return type

[**[]SessionMessage**](SessionMessage.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, text/event-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdMessagesPost

> SessionMessage ApiAmbientV1SessionsIdMessagesPost(ctx, id).SessionMessagePushRequest(sessionMessagePushRequest).Execute()

Push a message to a session



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	sessionMessagePushRequest := *openapiclient.NewSessionMessagePushRequest() // SessionMessagePushRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdMessagesPost(context.Background(), id).SessionMessagePushRequest(sessionMessagePushRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdMessagesPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdMessagesPost`: SessionMessage
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdMessagesPost`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdMessagesPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **sessionMessagePushRequest** | [**SessionMessagePushRequest**](SessionMessagePushRequest.md) |  | 

### Return type

[**SessionMessage**](SessionMessage.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdPatch

> Session ApiAmbientV1SessionsIdPatch(ctx, id).SessionPatchRequest(sessionPatchRequest).Execute()

Update an session

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	sessionPatchRequest := *openapiclient.NewSessionPatchRequest() // SessionPatchRequest | Updated session data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdPatch(context.Background(), id).SessionPatchRequest(sessionPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdPatch`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **sessionPatchRequest** | [**SessionPatchRequest**](SessionPatchRequest.md) | Updated session data | 

### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdStartPost

> Session ApiAmbientV1SessionsIdStartPost(ctx, id).Execute()

Start a session



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdStartPost(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdStartPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdStartPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdStartPost`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdStartPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdStatusPatch

> Session ApiAmbientV1SessionsIdStatusPatch(ctx, id).SessionStatusPatchRequest(sessionStatusPatchRequest).Execute()

Update session status fields



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	sessionStatusPatchRequest := *openapiclient.NewSessionStatusPatchRequest() // SessionStatusPatchRequest | Session status fields to update

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdStatusPatch(context.Background(), id).SessionStatusPatchRequest(sessionStatusPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdStatusPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdStatusPatch`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdStatusPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdStatusPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **sessionStatusPatchRequest** | [**SessionStatusPatchRequest**](SessionStatusPatchRequest.md) | Session status fields to update | 

### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsIdStopPost

> Session ApiAmbientV1SessionsIdStopPost(ctx, id).Execute()

Stop a session



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsIdStopPost(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsIdStopPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsIdStopPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsIdStopPost`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsIdStopPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1SessionsPost

> Session ApiAmbientV1SessionsPost(ctx).Session(session).Execute()

Create a new session

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	session := *openapiclient.NewSession("Name_example") // Session | Session data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1SessionsPost(context.Background()).Session(session).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1SessionsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1SessionsPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1SessionsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1SessionsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **session** | [**Session**](Session.md) | Session data | 

### Return type

[**Session**](Session.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1UsersGet

> UserList ApiAmbientV1UsersGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

Returns a list of users

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	page := int32(56) // int32 | Page number of record list when record list exceeds specified page size (optional) (default to 1)
	size := int32(56) // int32 | Maximum number of records to return (optional) (default to 100)
	search := "search_example" // string | Specifies the search criteria (optional)
	orderBy := "orderBy_example" // string | Specifies the order by criteria (optional)
	fields := "fields_example" // string | Supplies a comma-separated list of fields to be returned (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1UsersGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1UsersGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1UsersGet`: UserList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1UsersGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1UsersGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **page** | **int32** | Page number of record list when record list exceeds specified page size | [default to 1]
 **size** | **int32** | Maximum number of records to return | [default to 100]
 **search** | **string** | Specifies the search criteria | 
 **orderBy** | **string** | Specifies the order by criteria | 
 **fields** | **string** | Supplies a comma-separated list of fields to be returned | 

### Return type

[**UserList**](UserList.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1UsersIdGet

> User ApiAmbientV1UsersIdGet(ctx, id).Execute()

Get an user by id

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1UsersIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1UsersIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1UsersIdGet`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1UsersIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1UsersIdGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**User**](User.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1UsersIdPatch

> User ApiAmbientV1UsersIdPatch(ctx, id).UserPatchRequest(userPatchRequest).Execute()

Update an user

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	id := "id_example" // string | The id of record
	userPatchRequest := *openapiclient.NewUserPatchRequest() // UserPatchRequest | Updated user data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1UsersIdPatch(context.Background(), id).UserPatchRequest(userPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1UsersIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1UsersIdPatch`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1UsersIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1UsersIdPatchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **userPatchRequest** | [**UserPatchRequest**](UserPatchRequest.md) | Updated user data | 

### Return type

[**User**](User.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiAmbientV1UsersPost

> User ApiAmbientV1UsersPost(ctx).User(user).Execute()

Create a new user

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID"
)

func main() {
	user := *openapiclient.NewUser("Username_example", "Name_example") // User | User data

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DefaultAPI.ApiAmbientV1UsersPost(context.Background()).User(user).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientV1UsersPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientV1UsersPost`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientV1UsersPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientV1UsersPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **user** | [**User**](User.md) | User data | 

### Return type

[**User**](User.md)

### Authorization

[Bearer](../README.md#Bearer)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

