# Test

The `test` package provides a complete port of Node.js 18's experimental test runner (`node:test`) that works with Node.js 14+. It offers a comprehensive testing framework with minimal dependencies, supporting synchronous functions, Promise-based async functions, and callback-based functions for test execution.

## Package Information

- **Package Name**: test
- **Package Type**: npm
- **Language**: JavaScript (with TypeScript definitions)
- **Installation**: `npm install test`

## Core Imports

```javascript
import test, { describe, it, before, after, beforeEach, afterEach, run, mock } from "test";
```

For CommonJS:

```javascript
const test = require("test");
const { describe, it, before, after, beforeEach, afterEach, run, mock } = require("test");
```

## Basic Usage

```javascript
import test, { describe, it, beforeEach } from "test";

// Simple test
test("should add numbers correctly", (t) => {
  const result = 2 + 3;
  if (result !== 5) {
    throw new Error(`Expected 5, got ${result}`);
  }
});

// Test suite with hooks
describe("Calculator tests", () => {
  beforeEach(() => {
    console.log("Setting up test");
  });

  it("should multiply correctly", () => {
    const result = 4 * 5;
    if (result !== 20) {
      throw new Error(`Expected 20, got ${result}`);
    }
  });
});

// Async test
test("async operation", async (t) => {
  const result = await Promise.resolve(42);
  if (result !== 42) {
    throw new Error(`Expected 42, got ${result}`);
  }
});
```

## Architecture

The test package is built around several key components:

- **Core Test Functions**: `test`, `describe`, and `it` functions for organizing and running tests
- **Test Context System**: Context objects passed to test functions providing diagnostic and control capabilities
- **Hook System**: Lifecycle hooks (`before`, `after`, `beforeEach`, `afterEach`) for test setup and teardown
- **Test Runner**: Programmatic runner for executing test files with configurable options
- **Mocking System**: Comprehensive function and method mocking with call tracking and implementation control
- **CLI Tools**: Command-line utilities for various testing scenarios with TAP output support

## Capabilities

### Core Testing Functions

Primary test definition functions for creating individual tests and organizing them into suites. Supports multiple execution patterns and configuration options.

```javascript { .api }
function test(name: string, options: TestOptions, fn: TestFn): void;
function test(name: string, fn: TestFn): void;
function test(options: TestOptions, fn: TestFn): void;
function test(fn: TestFn): void;

function describe(name: string, options: TestOptions, fn: SuiteFn): void;
function describe(name: string, fn: SuiteFn): void;
function describe(options: TestOptions, fn: SuiteFn): void;
function describe(fn: SuiteFn): void;

function it(name: string, options: TestOptions, fn: ItFn): void;
function it(name: string, fn: ItFn): void;
function it(options: TestOptions, fn: ItFn): void;
function it(fn: ItFn): void;

interface TestOptions {
  concurrency?: boolean | number;
  skip?: boolean | string;
  todo?: boolean | string;
  timeout?: number;
  signal?: AbortSignal;
}
```

[Core Testing Functions](./core-testing.md)

### Test Lifecycle Hooks

Hook functions that run at specific points in the test lifecycle for setup and teardown operations.

```javascript { .api }
function before(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
function after(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
function beforeEach(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
function afterEach(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
```

[Test Lifecycle Hooks](./hooks.md)

### Test Runner

Programmatic test runner for executing test files with advanced configuration options including concurrency, timeouts, and custom reporters.

```javascript { .api }
function run(options?: RunOptions): TestsStream;

interface RunOptions {
  concurrency?: number;
  timeout?: number;
  signal?: AbortSignal;
  files?: string[];
  inspectPort?: number;
}
```

[Test Runner](./runner.md)

### Mocking System

Comprehensive mocking system for functions and object methods with call tracking, implementation control, and restoration capabilities.

```javascript { .api }
const mock: MockTracker;

interface MockTracker {
  fn(original?: Function, implementation?: Function, options?: MockOptions): MockFunctionContext;
  method(object: object, methodName: string, implementation?: Function, options?: MethodMockOptions): MockFunctionContext;
  getter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;
  setter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;
  reset(): void;
  restoreAll(): void;
}
```

[Mocking System](./mocking.md)

## Types

```javascript { .api }
type TestFn = (t: TestContext) => any | Promise<any>;
type SuiteFn = (t: SuiteContext) => void;
type ItFn = (t: ItContext) => any | Promise<any>;

interface TestContext {
  test(name: string, options: TestOptions, fn: TestFn): Promise<void>;
  test(name: string, fn: TestFn): Promise<void>;
  test(fn: TestFn): Promise<void>;
  diagnostic(message: string): void;
  skip(message?: string): void;
  todo(message?: string): void;
  signal: AbortSignal;
}

interface SuiteContext {
  signal: AbortSignal;
}

interface ItContext {
  signal: AbortSignal;
}
```