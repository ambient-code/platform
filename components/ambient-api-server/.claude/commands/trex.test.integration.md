# Run Integration Tests

Run integration tests against a real PostgreSQL database (provisioned automatically via testcontainers).

## Instructions

1. Build the binary to ensure everything compiles:
   ```bash
   make binary
   ```

2. Run integration tests:
   ```bash
   make test-integration
   ```

3. To run a specific test or subset:
   ```bash
   TESTFLAGS="-run TestDinosaurGet" make test-integration
   ```

4. Report results with pass/fail summary.

## How Integration Tests Work

- Database: Testcontainers automatically starts a PostgreSQL instance (no manual setup needed)
- Each test calls `test.RegisterIntegration(t)` which returns a `(*Helper, *openapi.APIClient)`
- The database is reset between tests via `helper.DBFactory.ResetDB()`
- Tests use gomega matchers (`Expect(...).To(...)`)

## Test Files

- `test/integration/` — Contains all integration test files

## Adding New Test Files

Follow the pattern in existing test files. Every test function should:
1. Call `h, client := test.RegisterIntegration(t)`
2. Create an account: `account := h.NewRandAccount()`
3. Get an authenticated context: `ctx := h.NewAuthenticatedContext(account)`
4. Use `client.DefaultAPI.Api...` methods for API calls
