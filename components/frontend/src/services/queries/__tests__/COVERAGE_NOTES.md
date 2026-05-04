# Integration Test Coverage Notes

## Port methods without hooks (known exceptions)

These port methods are intentionally not wrapped in React Query hooks.
They are called directly by components or other adapters, not through
the hook layer. A v2 adapter swap would still need manual verification
for these methods.

| Port | Method | Reason |
|------|--------|--------|
| `SessionsPort` | `saveToGoogleDrive` | Called directly by component action handlers |
| `GooglePort` | `getGoogleOAuthURL` | Returns a URL for redirect; no caching needed |
| `GerritPort` | `getGerritInstanceStatus` | Used for per-instance status checks in components |

## Streaming port (separate test location)

The `SessionEventsPort` (streaming) integration test lives in
`hooks/__tests__/integration-agui-stream.test.ts`, not in `queries/__tests__/`,
because `useAGUIStream` is a stateful hook (not React Query) located in `hooks/`.
The test uses `createSessionEventsAdapter(fakeApi)` with a `MockEventSource`
and real `processAGUIEvent` — no `vi.mock`. Covers all 3 port methods:
`createEventSource`, `sendMessage`, `interrupt`.

## Deprecated hooks (covered via paginated equivalents)

`useProjects()` and `useSessions()` are thin wrappers over
`useProjectsPaginated` / `useSessionsPaginated` that extract `.items`.
Both are deprecated. The paginated hooks have full integration tests
including `nextPage()` exercise.
