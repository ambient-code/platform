# Core Testing Functions

Primary test definition functions for creating individual tests and organizing them into suites. These functions support multiple execution patterns, configuration options, and provide the foundation for all test operations.

## Capabilities

### Test Function

The main test function for creating individual test cases. Supports multiple overloads for different usage patterns.

```javascript { .api }
/**
 * Creates and runs an individual test case
 * @param name - Test name (optional)
 * @param options - Test configuration options (optional)
 * @param fn - Test function to execute
 */
function test(name: string, options: TestOptions, fn: TestFn): void;
function test(name: string, fn: TestFn): void;
function test(options: TestOptions, fn: TestFn): void;
function test(fn: TestFn): void;

type TestFn = (t: TestContext) => any | Promise<any>;
```

**Usage Examples:**

```javascript
import test from "test";

// Named test
test("should calculate sum", (t) => {
  const result = 2 + 3;
  if (result !== 5) throw new Error("Math is broken");
});

// Test with options
test("slow operation", { timeout: 5000 }, async (t) => {
  await new Promise(resolve => setTimeout(resolve, 1000));
});

// Skipped test
test("not ready yet", { skip: "Feature not implemented" }, (t) => {
  // This won't run
});

// Anonymous test
test((t) => {
  t.diagnostic("Running anonymous test");
});
```

### Describe Function

Groups related tests into suites for better organization and shared setup/teardown.

```javascript { .api }
/**
 * Creates a test suite grouping related tests
 * @param name - Suite name (optional)
 * @param options - Suite configuration options (optional)
 * @param fn - Suite function containing tests and hooks
 */
function describe(name: string, options: TestOptions, fn: SuiteFn): void;
function describe(name: string, fn: SuiteFn): void;
function describe(options: TestOptions, fn: SuiteFn): void;
function describe(fn: SuiteFn): void;

type SuiteFn = (t: SuiteContext) => void;
```

**Usage Examples:**

```javascript
import { describe, it, beforeEach } from "test";

describe("User authentication", () => {
  beforeEach(() => {
    // Setup for each test in this suite
  });

  it("should login with valid credentials", () => {
    // Test implementation
  });

  it("should reject invalid credentials", () => {
    // Test implementation
  });
});

// Nested suites
describe("API endpoints", () => {
  describe("User endpoints", () => {
    it("should create user", () => {
      // Test implementation
    });
  });
});
```

### It Function

Creates individual test cases within describe blocks. Functionally identical to the test function but semantically used within suites.

```javascript { .api }
/**
 * Creates an individual test case within a suite
 * @param name - Test name (optional)
 * @param options - Test configuration options (optional)  
 * @param fn - Test function to execute
 */
function it(name: string, options: TestOptions, fn: ItFn): void;
function it(name: string, fn: ItFn): void;
function it(options: TestOptions, fn: ItFn): void;
function it(fn: ItFn): void;

type ItFn = (t: ItContext) => any | Promise<any>;
```

**Usage Examples:**

```javascript
import { describe, it } from "test";

describe("Calculator", () => {
  it("should add two numbers", (t) => {
    const result = add(2, 3);
    if (result !== 5) throw new Error("Addition failed");
  });

  it("handles negative numbers", { timeout: 1000 }, (t) => {
    const result = add(-1, 1);
    if (result !== 0) throw new Error("Negative addition failed");
  });

  it("TODO: should handle decimals", { todo: true }, (t) => {
    // Not implemented yet
  });
});
```

## Test Options

```javascript { .api }
interface TestOptions {
  /** Number of tests that can run concurrently. Default: 1 */
  concurrency?: boolean | number;
  /** Skip test with optional reason. Default: false */
  skip?: boolean | string;
  /** Mark test as TODO with optional reason. Default: false */
  todo?: boolean | string;
  /** Test timeout in milliseconds. Default: Infinity */
  timeout?: number;
  /** AbortSignal for test cancellation */
  signal?: AbortSignal;
}
```

## Context Objects

### TestContext

Context object passed to test functions providing diagnostic and control capabilities.

```javascript { .api }
interface TestContext {
  /** Create subtests within the current test */
  test(name: string, options: TestOptions, fn: TestFn): Promise<void>;
  test(name: string, fn: TestFn): Promise<void>;
  test(fn: TestFn): Promise<void>;
  /** Write diagnostic information to test output */
  diagnostic(message: string): void;
  /** Mark current test as skipped */
  skip(message?: string): void;
  /** Mark current test as TODO */
  todo(message?: string): void;
  /** AbortSignal for test cancellation */
  signal: AbortSignal;
}
```

**Usage Examples:**

```javascript
test("test with subtests", async (t) => {
  t.diagnostic("Starting parent test");
  
  await t.test("subtest 1", (st) => {
    // First subtest
  });
  
  await t.test("subtest 2", { timeout: 500 }, (st) => {
    // Second subtest with timeout
  });
});

test("conditional skip", (t) => {
  if (!process.env.API_KEY) {
    t.skip("API key not provided");
    return;
  }
  // Test implementation
});
```

### SuiteContext

Context object passed to describe functions.

```javascript { .api }
interface SuiteContext {
  /** AbortSignal for suite cancellation */
  signal: AbortSignal;
}
```

### ItContext

Context object passed to it functions.

```javascript { .api }
interface ItContext {
  /** AbortSignal for test cancellation */
  signal: AbortSignal;
}
```

## Error Handling

Tests can fail by:
- Throwing any error or exception
- Returning a rejected Promise
- Timing out (when timeout option is set)
- Being aborted via AbortSignal

```javascript
test("error handling examples", (t) => {
  // Explicit error
  throw new Error("Test failed");
  
  // Assertion-style
  if (result !== expected) {
    throw new Error(`Expected ${expected}, got ${result}`);
  }
});

test("async error handling", async (t) => {
  // Rejected promise
  await Promise.reject(new Error("Async operation failed"));
  
  // Async assertion
  const result = await fetchData();
  if (!result) {
    throw new Error("No data received");
  }
});
```