# Mocking System

Comprehensive mocking system for functions and object methods with call tracking, implementation control, and restoration capabilities. The mocking system provides fine-grained control over function behavior during testing.

## Capabilities

### MockTracker

The main mocking interface accessed via the `mock` property. Provides methods for creating and managing various types of mocks.

```javascript { .api }
/**
 * Main mocking interface for creating and managing mocks
 */
const mock: MockTracker;

interface MockTracker {
  /** Create a function mock */
  fn(original?: Function, implementation?: Function, options?: MockOptions): MockFunctionContext;
  /** Mock an object method */
  method(object: object, methodName: string, implementation?: Function, options?: MethodMockOptions): MockFunctionContext;
  /** Mock an object getter */
  getter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;
  /** Mock an object setter */
  setter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;
  /** Reset all mock call tracking data */
  reset(): void;
  /** Restore all mocked functions to their originals */
  restoreAll(): void;
}
```

### Function Mocking

Create mock functions for testing function behavior and call patterns.

```javascript { .api }
/**
 * Create a function mock
 * @param original - Original function to mock (optional)
 * @param implementation - Mock implementation (optional)
 * @param options - Mock configuration options
 * @returns MockFunctionContext for controlling and inspecting the mock
 */
fn(original?: Function, implementation?: Function, options?: MockOptions): MockFunctionContext;

interface MockOptions {
  /** Number of times mock should be active before restoring */
  times?: number;
}
```

**Usage Examples:**

```javascript
import { mock } from "test";

// Basic function mock
const mockFn = mock.fn();
mockFn('hello', 'world');
console.log(mockFn.callCount()); // 1

// Mock with implementation
const add = mock.fn(null, (a, b) => a + b);
console.log(add(2, 3)); // 5

// Mock existing function
const originalFetch = fetch;
const mockFetch = mock.fn(originalFetch, async () => ({
  json: async () => ({ success: true })
}));

// Temporary mock (auto-restores after 2 calls)
const tempMock = mock.fn(console.log, () => {}, { times: 2 });
tempMock('call 1'); // mock implementation
tempMock('call 2'); // mock implementation  
tempMock('call 3'); // original console.log
```

### Method Mocking

Mock methods on existing objects while preserving the object structure.

```javascript { .api }
/**
 * Mock an object method
 * @param object - Target object containing the method
 * @param methodName - Name of the method to mock
 * @param implementation - Mock implementation (optional)
 * @param options - Mock configuration options
 * @returns MockFunctionContext for controlling and inspecting the mock
 */
method(object: object, methodName: string, implementation?: Function, options?: MethodMockOptions): MockFunctionContext;

interface MethodMockOptions extends MockOptions {
  /** Mock as getter property */
  getter?: boolean;
  /** Mock as setter property */
  setter?: boolean;
}
```

**Usage Examples:**

```javascript
import { mock } from "test";

// Mock object method
const user = {
  name: 'Alice',
  getName() { return this.name; }
};

const mockGetName = mock.method(user, 'getName', function() {
  return 'Mocked Name';
});

console.log(user.getName()); // 'Mocked Name'
console.log(mockGetName.callCount()); // 1

// Mock with original function access
const fs = require('fs');
const mockReadFile = mock.method(fs, 'readFileSync', (path) => {
  if (path === 'test.txt') return 'mocked content';
  // Call original for other files
  return mockReadFile.original(path);
});

// Mock getter/setter
const config = {
  _timeout: 1000,
  get timeout() { return this._timeout; },
  set timeout(value) { this._timeout = value; }
};

mock.method(config, 'timeout', () => 5000, { getter: true });
console.log(config.timeout); // 5000
```

### Getter and Setter Mocking

Specialized methods for mocking property getters and setters.

```javascript { .api }
/**
 * Mock an object getter
 * @param object - Target object
 * @param methodName - Property name
 * @param implementation - Getter implementation
 * @param options - Mock options
 * @returns MockFunctionContext
 */
getter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;

/**
 * Mock an object setter  
 * @param object - Target object
 * @param methodName - Property name
 * @param implementation - Setter implementation
 * @param options - Mock options
 * @returns MockFunctionContext
 */
setter(object: object, methodName: string, implementation?: Function, options?: MockOptions): MockFunctionContext;
```

**Usage Examples:**

```javascript
import { mock } from "test";

const obj = {
  _value: 42,
  get value() { return this._value; },
  set value(v) { this._value = v; }
};

// Mock getter
const mockGetter = mock.getter(obj, 'value', () => 999);
console.log(obj.value); // 999

// Mock setter
const mockSetter = mock.setter(obj, 'value', function(v) {
  console.log(`Setting value to ${v}`);
  this._value = v * 2;
});

obj.value = 10; // Logs: "Setting value to 10"
console.log(obj._value); // 20
```

## MockFunctionContext

Context object returned by all mock creation methods, providing control and inspection capabilities.

```javascript { .api }
interface MockFunctionContext {
  /** Array of all function calls (read-only) */
  calls: CallRecord[];
  /** Get the total number of calls */
  callCount(): number;
  /** Change the mock implementation */
  mockImplementation(implementation: Function): void;
  /** Set implementation for a specific call number */
  mockImplementationOnce(implementation: Function, onCall?: number): void;
  /** Restore the original function */
  restore(): void;
}

interface CallRecord {
  arguments: any[];
  result?: any;
  error?: Error;
  target?: any;
  this?: any;
}
```

**Usage Examples:**

```javascript
import { mock } from "test";

const mockFn = mock.fn();

// Make some calls
mockFn('arg1', 'arg2');
mockFn(123);
mockFn();

// Inspect calls
console.log(mockFn.callCount()); // 3
console.log(mockFn.calls[0].arguments); // ['arg1', 'arg2']
console.log(mockFn.calls[1].arguments); // [123]

// Change implementation
mockFn.mockImplementation((x) => x * 2);
console.log(mockFn(5)); // 10

// One-time implementation
mockFn.mockImplementationOnce(() => 'special', 4);
mockFn(); // 'special' (5th call)
mockFn(); // back to x * 2 implementation

// Restore original
mockFn.restore();
```

### Call Inspection

Detailed inspection of mock function calls:

```javascript
import { mock } from "test";

const calculator = {
  add(a, b) { return a + b; }
};

const mockAdd = mock.method(calculator, 'add');

// Make calls
calculator.add(2, 3);
calculator.add(10, 5);

// Inspect calls
const calls = mockAdd.calls;

calls.forEach((call, index) => {
  console.log(`Call ${index + 1}:`);
  console.log(`  Arguments: ${call.arguments}`);
  console.log(`  Result: ${call.result}`);
  console.log(`  This context:`, call.this);
});

// Check specific calls
if (mockAdd.callCount() >= 2) {
  const secondCall = calls[1];
  if (secondCall.arguments[0] === 10 && secondCall.arguments[1] === 5) {
    console.log('Second call was add(10, 5)');
  }
}
```

## Global Mock Management

### Reset All Mocks

Clear call tracking data for all mocks without restoring implementations:

```javascript { .api }
/**
 * Reset all mock call tracking data
 */
reset(): void;
```

**Usage Examples:**

```javascript
import { mock } from "test";

const mockFn1 = mock.fn();
const mockFn2 = mock.fn();

mockFn1('test');
mockFn2('test');

console.log(mockFn1.callCount()); // 1
console.log(mockFn2.callCount()); // 1

mock.reset();

console.log(mockFn1.callCount()); // 0
console.log(mockFn2.callCount()); // 0
// Functions still mocked, just call history cleared
```

### Restore All Mocks

Restore all mocked functions to their original implementations:

```javascript { .api }
/**
 * Restore all mocked functions to their originals
 */
restoreAll(): void;
```

**Usage Examples:**

```javascript
import { mock } from "test";

const obj = {
  method1() { return 'original1'; },
  method2() { return 'original2'; }
};

mock.method(obj, 'method1', () => 'mocked1');
mock.method(obj, 'method2', () => 'mocked2');

console.log(obj.method1()); // 'mocked1'
console.log(obj.method2()); // 'mocked2'

mock.restoreAll();

console.log(obj.method1()); // 'original1'
console.log(obj.method2()); // 'original2'
```

## Advanced Patterns

### Conditional Mocking

Mock functions that behave differently based on arguments:

```javascript
import { mock } from "test";

const mockFetch = mock.fn(fetch, async (url, options) => {
  if (url.includes('/api/users')) {
    return { json: async () => [{ id: 1, name: 'Test User' }] };
  }
  if (url.includes('/api/error')) {
    throw new Error('Network error');
  }
  // Call original fetch for other URLs
  return fetch(url, options);
});
```

### Spy Pattern

Monitor function calls without changing behavior:

```javascript
import { mock } from "test";

const logger = {
  log(message) {
    console.log(`[LOG] ${message}`);
  }
};

// Spy on logger.log (keep original behavior)
const logSpy = mock.method(logger, 'log', function(message) {
  // Call original implementation
  return logSpy.original.call(this, message);
});

logger.log('Hello'); // Still logs to console
console.log(logSpy.callCount()); // 1
console.log(logSpy.calls[0].arguments[0]); // 'Hello'
```

### Mock Chaining

Create complex mock setups:

```javascript
import { mock } from "test";

const api = {
  get() { return Promise.resolve({ data: 'real data' }); },
  post() { return Promise.resolve({ success: true }); }
};

// Chain multiple mocks
mock.method(api, 'get', () => Promise.resolve({ data: 'mock data' }))
mock.method(api, 'post', () => Promise.resolve({ success: false }));

// Test with mocked API
test('API interactions', async (t) => {
  const getData = await api.get();
  const postResult = await api.post();
  
  if (getData.data !== 'mock data') {
    throw new Error('GET mock failed');
  }
  if (postResult.success !== false) {
    throw new Error('POST mock failed');
  }
  
  // Clean up
  mock.restoreAll();
});
```

### Error Simulation

Mock functions to simulate error conditions:

```javascript
import { mock } from "test";

const fileSystem = {
  readFile(path) {
    // Original implementation
    return fs.readFileSync(path, 'utf8');
  }
};

// Mock to simulate file not found
const mockReadFile = mock.method(fileSystem, 'readFile', (path) => {
  if (path === 'nonexistent.txt') {
    const error = new Error('ENOENT: no such file or directory');
    error.code = 'ENOENT';
    throw error;
  }
  return 'file contents';
});

test('error handling', (t) => {
  try {
    fileSystem.readFile('nonexistent.txt');
    throw new Error('Should have thrown');
  } catch (error) {
    if (error.code !== 'ENOENT') {
      throw new Error('Wrong error type');
    }
  }
  
  // Verify the error was tracked
  if (mockReadFile.calls[0].error.code !== 'ENOENT') {
    throw new Error('Error not tracked correctly');
  }
});
```

## Best Practices

1. **Clean up mocks**: Always restore mocks after tests
2. **Use spies for monitoring**: Keep original behavior when you just need to track calls
3. **Mock at the right level**: Mock dependencies, not implementation details
4. **Test mock behavior**: Verify mocks are called with expected arguments
5. **Avoid over-mocking**: Only mock what's necessary for the test

```javascript
import { mock, afterEach } from "test";

// Clean up after each test
afterEach(() => {
  mock.restoreAll();
});

test('user service', (t) => {
  const database = {
    findUser: (id) => ({ id, name: 'Real User' })
  };

  // Mock external dependency
  const mockFindUser = mock.method(database, 'findUser', 
    (id) => ({ id, name: 'Test User' })
  );

  const userService = new UserService(database);
  const user = userService.getUser(123);

  // Test the result
  if (user.name !== 'Test User') {
    throw new Error('Service failed');
  }

  // Verify mock was called correctly
  if (mockFindUser.callCount() !== 1) {
    throw new Error('Database not called');
  }
  if (mockFindUser.calls[0].arguments[0] !== 123) {
    throw new Error('Wrong user ID passed');
  }
});
```