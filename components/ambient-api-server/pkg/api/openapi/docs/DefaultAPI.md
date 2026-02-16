# \DefaultAPI

All URIs are relative to *http://localhost:8000*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiAmbientApiServerV1ProjectSettingsGet**](DefaultAPI.md#ApiAmbientApiServerV1ProjectSettingsGet) | **Get** /api/ambient-api-server/v1/project_settings | Returns a list of project settings
[**ApiAmbientApiServerV1ProjectSettingsIdDelete**](DefaultAPI.md#ApiAmbientApiServerV1ProjectSettingsIdDelete) | **Delete** /api/ambient-api-server/v1/project_settings/{id} | Delete a project settings by id
[**ApiAmbientApiServerV1ProjectSettingsIdGet**](DefaultAPI.md#ApiAmbientApiServerV1ProjectSettingsIdGet) | **Get** /api/ambient-api-server/v1/project_settings/{id} | Get a project settings by id
[**ApiAmbientApiServerV1ProjectSettingsIdPatch**](DefaultAPI.md#ApiAmbientApiServerV1ProjectSettingsIdPatch) | **Patch** /api/ambient-api-server/v1/project_settings/{id} | Update a project settings
[**ApiAmbientApiServerV1ProjectSettingsPost**](DefaultAPI.md#ApiAmbientApiServerV1ProjectSettingsPost) | **Post** /api/ambient-api-server/v1/project_settings | Create a new project settings
[**ApiAmbientApiServerV1ProjectsGet**](DefaultAPI.md#ApiAmbientApiServerV1ProjectsGet) | **Get** /api/ambient-api-server/v1/projects | Returns a list of projects
[**ApiAmbientApiServerV1ProjectsIdDelete**](DefaultAPI.md#ApiAmbientApiServerV1ProjectsIdDelete) | **Delete** /api/ambient-api-server/v1/projects/{id} | Delete a project by id
[**ApiAmbientApiServerV1ProjectsIdGet**](DefaultAPI.md#ApiAmbientApiServerV1ProjectsIdGet) | **Get** /api/ambient-api-server/v1/projects/{id} | Get a project by id
[**ApiAmbientApiServerV1ProjectsIdPatch**](DefaultAPI.md#ApiAmbientApiServerV1ProjectsIdPatch) | **Patch** /api/ambient-api-server/v1/projects/{id} | Update a project
[**ApiAmbientApiServerV1ProjectsPost**](DefaultAPI.md#ApiAmbientApiServerV1ProjectsPost) | **Post** /api/ambient-api-server/v1/projects | Create a new project
[**ApiAmbientApiServerV1SessionsGet**](DefaultAPI.md#ApiAmbientApiServerV1SessionsGet) | **Get** /api/ambient-api-server/v1/sessions | Returns a list of sessions
[**ApiAmbientApiServerV1SessionsIdGet**](DefaultAPI.md#ApiAmbientApiServerV1SessionsIdGet) | **Get** /api/ambient-api-server/v1/sessions/{id} | Get an session by id
[**ApiAmbientApiServerV1SessionsIdPatch**](DefaultAPI.md#ApiAmbientApiServerV1SessionsIdPatch) | **Patch** /api/ambient-api-server/v1/sessions/{id} | Update an session
[**ApiAmbientApiServerV1SessionsIdStartPost**](DefaultAPI.md#ApiAmbientApiServerV1SessionsIdStartPost) | **Post** /api/ambient-api-server/v1/sessions/{id}/start | Start a session
[**ApiAmbientApiServerV1SessionsIdStatusPatch**](DefaultAPI.md#ApiAmbientApiServerV1SessionsIdStatusPatch) | **Patch** /api/ambient-api-server/v1/sessions/{id}/status | Update session status fields
[**ApiAmbientApiServerV1SessionsIdStopPost**](DefaultAPI.md#ApiAmbientApiServerV1SessionsIdStopPost) | **Post** /api/ambient-api-server/v1/sessions/{id}/stop | Stop a session
[**ApiAmbientApiServerV1SessionsPost**](DefaultAPI.md#ApiAmbientApiServerV1SessionsPost) | **Post** /api/ambient-api-server/v1/sessions | Create a new session
[**ApiAmbientApiServerV1UsersGet**](DefaultAPI.md#ApiAmbientApiServerV1UsersGet) | **Get** /api/ambient-api-server/v1/users | Returns a list of users
[**ApiAmbientApiServerV1UsersIdGet**](DefaultAPI.md#ApiAmbientApiServerV1UsersIdGet) | **Get** /api/ambient-api-server/v1/users/{id} | Get an user by id
[**ApiAmbientApiServerV1UsersIdPatch**](DefaultAPI.md#ApiAmbientApiServerV1UsersIdPatch) | **Patch** /api/ambient-api-server/v1/users/{id} | Update an user
[**ApiAmbientApiServerV1UsersPost**](DefaultAPI.md#ApiAmbientApiServerV1UsersPost) | **Post** /api/ambient-api-server/v1/users | Create a new user



## ApiAmbientApiServerV1ProjectSettingsGet

> ProjectSettingsList ApiAmbientApiServerV1ProjectSettingsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectSettingsGet`: ProjectSettingsList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectSettingsGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectSettingsIdDelete

> ApiAmbientApiServerV1ProjectSettingsIdDelete(ctx, id).Execute()

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
	r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdDelete``: %v\n", err)
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

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectSettingsIdDeleteRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectSettingsIdGet

> ProjectSettings ApiAmbientApiServerV1ProjectSettingsIdGet(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectSettingsIdGet`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectSettingsIdGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectSettingsIdPatch

> ProjectSettings ApiAmbientApiServerV1ProjectSettingsIdPatch(ctx, id).ProjectSettingsPatchRequest(projectSettingsPatchRequest).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdPatch(context.Background(), id).ProjectSettingsPatchRequest(projectSettingsPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectSettingsIdPatch`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectSettingsIdPatchRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectSettingsPost

> ProjectSettings ApiAmbientApiServerV1ProjectSettingsPost(ctx).ProjectSettings(projectSettings).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsPost(context.Background()).ProjectSettings(projectSettings).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectSettingsPost`: ProjectSettings
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectSettingsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectSettingsPostRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectsGet

> ProjectList ApiAmbientApiServerV1ProjectsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectsGet`: ProjectList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectsGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectsIdDelete

> ApiAmbientApiServerV1ProjectsIdDelete(ctx, id).Execute()

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
	r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectsIdDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectsIdDelete``: %v\n", err)
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

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectsIdDeleteRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectsIdGet

> Project ApiAmbientApiServerV1ProjectsIdGet(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectsIdGet`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectsIdGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectsIdPatch

> Project ApiAmbientApiServerV1ProjectsIdPatch(ctx, id).ProjectPatchRequest(projectPatchRequest).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectsIdPatch(context.Background(), id).ProjectPatchRequest(projectPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectsIdPatch`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectsIdPatchRequest struct via the builder pattern


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


## ApiAmbientApiServerV1ProjectsPost

> Project ApiAmbientApiServerV1ProjectsPost(ctx).Project(project).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(context.Background()).Project(project).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1ProjectsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1ProjectsPost`: Project
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1ProjectsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1ProjectsPostRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsGet

> SessionList ApiAmbientApiServerV1SessionsGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsGet`: SessionList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsIdGet

> Session ApiAmbientApiServerV1SessionsIdGet(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsIdGet`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsIdGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsIdPatch

> Session ApiAmbientApiServerV1SessionsIdPatch(ctx, id).SessionPatchRequest(sessionPatchRequest).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsIdPatch(context.Background(), id).SessionPatchRequest(sessionPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsIdPatch`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsIdPatchRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsIdStartPost

> Session ApiAmbientApiServerV1SessionsIdStartPost(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsIdStartPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsIdStartPostRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsIdStatusPatch

> Session ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, id).SessionStatusPatchRequest(sessionStatusPatchRequest).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(context.Background(), id).SessionStatusPatchRequest(sessionStatusPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsIdStatusPatch`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsIdStatusPatchRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsIdStopPost

> Session ApiAmbientApiServerV1SessionsIdStopPost(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsIdStopPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsIdStopPostRequest struct via the builder pattern


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


## ApiAmbientApiServerV1SessionsPost

> Session ApiAmbientApiServerV1SessionsPost(ctx).Session(session).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1SessionsPost(context.Background()).Session(session).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1SessionsPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1SessionsPost`: Session
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1SessionsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1SessionsPostRequest struct via the builder pattern


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


## ApiAmbientApiServerV1UsersGet

> UserList ApiAmbientApiServerV1UsersGet(ctx).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1UsersGet(context.Background()).Page(page).Size(size).Search(search).OrderBy(orderBy).Fields(fields).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1UsersGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1UsersGet`: UserList
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1UsersGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1UsersGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1UsersIdGet

> User ApiAmbientApiServerV1UsersIdGet(ctx, id).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1UsersIdGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1UsersIdGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1UsersIdGet`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1UsersIdGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1UsersIdGetRequest struct via the builder pattern


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


## ApiAmbientApiServerV1UsersIdPatch

> User ApiAmbientApiServerV1UsersIdPatch(ctx, id).UserPatchRequest(userPatchRequest).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1UsersIdPatch(context.Background(), id).UserPatchRequest(userPatchRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1UsersIdPatch``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1UsersIdPatch`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1UsersIdPatch`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The id of record | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1UsersIdPatchRequest struct via the builder pattern


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


## ApiAmbientApiServerV1UsersPost

> User ApiAmbientApiServerV1UsersPost(ctx).User(user).Execute()

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
	resp, r, err := apiClient.DefaultAPI.ApiAmbientApiServerV1UsersPost(context.Background()).User(user).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DefaultAPI.ApiAmbientApiServerV1UsersPost``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiAmbientApiServerV1UsersPost`: User
	fmt.Fprintf(os.Stdout, "Response from `DefaultAPI.ApiAmbientApiServerV1UsersPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiAmbientApiServerV1UsersPostRequest struct via the builder pattern


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

