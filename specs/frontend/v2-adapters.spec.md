# Frontend v2 Adapter Specification

## Purpose

The frontend adapter layer SHALL support a second generation of adapters (v2) that consume the platform's REST API server instead of the legacy Kubernetes-backed backend. v2 adapters implement the same port interfaces defined in [api-adapter.spec.md](api-adapter.spec.md), enabling incremental per-domain migration without changes to React components or React Query hooks.

Real-time and streaming operations (session messages, WebSocket connections) are out of scope for this spec. v2 adapters cover CRUD and lifecycle operations only.

## Requirements

### Requirement: Response Transformation

v2 adapters SHALL transform API server responses into canonical frontend types. The canonical types preserve the existing structure that components consume.

#### Scenario: Session response maps to canonical AgenticSession

- GIVEN an API server response for a session resource
- WHEN the v2 adapter transforms the response
- THEN the result conforms to the canonical `AgenticSession` type with `metadata`, `spec`, and `status` sub-objects
- AND `metadata` includes `name`, `namespace`, `uid`, `creationTimestamp`, `labels`, `annotations`
- AND `spec` includes `initialPrompt`, `displayName`, `llmSettings` (as a typed object with `model`, `temperature`, `maxTokens`), `repos` (as a typed array), `timeout`, `environmentVariables` (as a typed record), and all other spec fields
- AND `status` includes `phase`, `startTime`, `completionTime`, `conditions` (as a typed array), `reconciledRepos` (as a typed array), `reconciledWorkflow` (as a typed object), `sdkSessionId`, `sdkRestartCount`
- AND all field names use camelCase

#### Scenario: Project response maps to canonical Project

- GIVEN an API server response for a project resource
- WHEN the v2 adapter transforms the response
- THEN the result conforms to the canonical `Project` type with `name`, `displayName`, `description`, `status`, `labels`, `annotations`, `creationTimestamp`, `uid`
- AND `labels` and `annotations` are `Record<string, string>`, not serialized strings

#### Scenario: Serialized collection fields are parsed

- GIVEN an API server response containing collection fields serialized as JSON strings
- WHEN the v2 adapter transforms the response
- THEN those fields are parsed into their typed representations (arrays, records, nested objects)
- AND the consumer receives the same typed structures as from a v1 adapter

#### Scenario: Missing canonical fields have defaults

- GIVEN an API server response that lacks fields present in the canonical type
- WHEN the v2 adapter transforms the response
- THEN those fields SHALL have documented default values
- AND the canonical type contract is satisfied without runtime errors

### Requirement: Pagination Contract

v2 adapters SHALL produce `PaginatedResult<T>` that is indistinguishable from v1 pagination at the consumer level, regardless of the underlying pagination model.

#### Scenario: Paginated list returns correct metadata

- GIVEN a v2 adapter listing sessions
- WHEN the consumer receives a `PaginatedResult<AgenticSession>`
- THEN `totalCount` reflects the total number of matching records across all pages
- AND `hasMore` is `true` when additional pages exist, `false` otherwise
- AND `nextPage()` returns the next page of results when called
- AND `nextPage` is `undefined` when no more pages exist

#### Scenario: Consumer code cannot distinguish v1 from v2 pagination

- GIVEN a hook consuming `PaginatedResult<T>` from either a v1 or v2 adapter
- WHEN the hook iterates pages using `nextPage()`
- THEN the behavior is identical regardless of which adapter version produced the result
- AND the consumer never observes the underlying pagination mechanism (offset-based or page-based)

#### Scenario: Pagination input parameters are transparent

- GIVEN a consumer passing `PaginationParams` (with `limit` and `offset`) to a v2 adapter
- WHEN the v2 adapter receives the parameters
- THEN the adapter translates them to the underlying pagination model
- AND the consumer never needs to change its pagination parameters when switching from v1 to v2

### Requirement: Error Normalization

v2 adapters SHALL normalize API server errors into the canonical `ApiError` type.

#### Scenario: API server error maps to canonical error

- GIVEN an API server error response
- WHEN the v2 adapter processes the error
- THEN the canonical `ApiError.error` field contains a human-readable reason
- AND `ApiError.code` contains an error code when available
- AND backend-specific error metadata (operation IDs, hrefs, kind strings, status codes) does not appear in the canonical error type

#### Scenario: Error type consistency

- GIVEN a consumer with error handling logic for `ApiError`
- WHEN the consumer handles errors from a v2 adapter
- THEN the same error handling code works for both v1 and v2 adapter errors without modification

### Requirement: Migration Data Integrity

When a domain is migrated from v1 to v2, consumers SHALL NOT receive stale data cached from the previous adapter version.

#### Scenario: Domain migration does not serve stale data

- GIVEN sessions previously served by a v1 adapter with cached query results
- WHEN the sessions domain is migrated to a v2 adapter
- THEN the v2 adapter's responses populate a separate cache from v1
- AND consumers receive fresh data from the v2 adapter, not stale v1 cache entries

### Requirement: Auth Transparency

v2 adapters SHALL NOT require consumers to provide, manage, or be aware of authentication tokens.

#### Scenario: Port consumers never handle auth

- GIVEN a React Query hook calling a v2 adapter through a port interface
- WHEN the hook invokes any port method
- THEN the hook provides no authentication parameters
- AND authentication is handled transparently below the port boundary
- AND the auth mechanism is identical in behavior to v1 adapters from the consumer's perspective

### Requirement: Adapter Coexistence

v2 adapters SHALL coexist with v1 adapters without observable conflict. This extends the "Incremental Adoption" requirement in [api-adapter.spec.md](api-adapter.spec.md).

#### Scenario: Mixed v1 and v2 adapters operate simultaneously

- GIVEN a v2 adapter for sessions and a v1 adapter for projects
- WHEN both are active simultaneously
- THEN session operations use the v2 adapter and return data from the API server
- AND project operations use the v1 adapter and return data from the legacy backend
- AND no cross-domain interference occurs (cache, error handling, or auth)

#### Scenario: Migration is per-domain

- GIVEN the set of port domains defined in [api-adapter.spec.md](api-adapter.spec.md)
- WHEN a subset of domains has v2 adapters
- THEN each domain operates independently on its assigned adapter version
- AND migrating one domain does not require migrating any other domain

### Requirement: Request Transformation

v2 adapters SHALL transform canonical frontend request types into API server request formats. Consumers continue to pass the same request types regardless of which adapter version is active.

#### Scenario: Session creation request maps to API server format

- GIVEN a `CreateAgenticSessionRequest` with nested camelCase fields (e.g., `llmSettings` as a typed object, `repos` as a typed array, `environmentVariables` as a record)
- WHEN the v2 adapter submits the creation request
- THEN nested fields are flattened to the API server's expected format
- AND collection fields are serialized appropriately
- AND the consumer's request type is unchanged from v1

#### Scenario: Session update request maps to API server format

- GIVEN a session update with canonical field names
- WHEN the v2 adapter submits the patch
- THEN only changed fields are sent
- AND field names and structure match the API server's expectation

#### Scenario: Project mutation requests map to API server format

- GIVEN a `CreateProjectRequest` or `UpdateProjectRequest` with canonical field names
- WHEN the v2 adapter submits the request
- THEN the canonical request is transformed to the API server format
- AND collection fields (labels, annotations) are serialized

#### Scenario: Request fields without API server equivalents are handled gracefully

- GIVEN a canonical request type containing fields that have no API server equivalent
- WHEN the v2 adapter processes the request
- THEN those fields are silently dropped
- AND the remaining fields are submitted successfully
- AND no runtime error occurs

### Requirement: Unsupported Operations

v2 adapters SHALL declare a strategy for port methods that have no API server equivalent. Not all port methods can be backed by the v2 API server initially.

#### Scenario: Unsupported session operations produce a clear error

- GIVEN a port method that has no v2 implementation (e.g., a Kubernetes-only operation)
- WHEN a consumer calls that method through a v2 adapter
- THEN the adapter SHALL throw a documented error indicating the operation is not available
- AND the error type is consistent with the canonical error contract

#### Scenario: Unsupported project operations produce a clear error

- GIVEN a project port method that operates on domain-specific sub-resources not modeled in the API server
- WHEN a consumer calls that method through a v2 adapter
- THEN the adapter SHALL throw a documented error

#### Scenario: Partial migration is supported

- GIVEN a domain where some port methods are supported in v2 and others are not
- WHEN the v2 adapter is active
- THEN supported operations use the v2 API server
- AND unsupported operations produce a documented error
- AND the adapter documents which operations are not yet available
