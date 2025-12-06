# Test Lifecycle Hooks

Hook functions that run at specific points in the test lifecycle for setup and teardown operations. These hooks provide a clean way to prepare test environments and clean up resources.

## Capabilities

### Before Hook

Runs once before all tests in the current suite. Ideal for expensive setup operations that can be shared across multiple tests.

```javascript { .api }
/**
 * Runs once before all tests in the current suite
 * @param fn - Setup function to execute
 * @param options - Optional hook configuration
 */
function before(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
```

**Usage Examples:**

```javascript
import { describe, it, before } from "test";

describe("Database tests", () => {
  before(async () => {
    // Setup database connection
    await connectToDatabase();
    await runMigrations();
  });

  it("should create user", () => {
    // Test implementation
  });

  it("should update user", () => {
    // Test implementation
  });
});

// Top-level before hook
before(() => {
  console.log("Starting all tests");
  process.env.NODE_ENV = "test";
});
```

### After Hook

Runs once after all tests in the current suite have completed. Used for cleanup operations that should happen regardless of test success or failure.

```javascript { .api }
/**
 * Runs once after all tests in the current suite
 * @param fn - Cleanup function to execute
 * @param options - Optional hook configuration
 */
function after(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
```

**Usage Examples:**

```javascript
import { describe, it, before, after } from "test";

describe("File system tests", () => {
  before(() => {
    // Create test directory
    fs.mkdirSync('./test-files');
  });

  after(() => {
    // Clean up test directory
    fs.rmSync('./test-files', { recursive: true });
  });

  it("should create file", () => {
    // Test implementation
  });
});

// Top-level after hook
after(async () => {
  console.log("All tests completed");
  await cleanup();
});
```

### BeforeEach Hook

Runs before each individual test. Perfect for ensuring each test starts with a clean, predictable state.

```javascript { .api }
/**
 * Runs before each individual test
 * @param fn - Setup function to execute before each test
 * @param options - Optional hook configuration
 */
function beforeEach(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
```

**Usage Examples:**

```javascript
import { describe, it, beforeEach } from "test";

describe("Calculator tests", () => {
  let calculator;

  beforeEach(() => {
    // Fresh calculator instance for each test
    calculator = new Calculator();
    calculator.reset();
  });

  it("should add numbers", () => {
    const result = calculator.add(2, 3);
    if (result !== 5) throw new Error("Addition failed");
  });

  it("should subtract numbers", () => {
    const result = calculator.subtract(5, 3);
    if (result !== 2) throw new Error("Subtraction failed");
  });
});

// Async beforeEach
describe("API tests", () => {
  beforeEach(async () => {
    await resetDatabase();
    await seedTestData();
  });

  it("should fetch users", async () => {
    // Test implementation
  });
});
```

### AfterEach Hook

Runs after each individual test completes. Used for cleaning up test-specific resources and ensuring tests don't interfere with each other.

```javascript { .api }
/**
 * Runs after each individual test
 * @param fn - Cleanup function to execute after each test
 * @param options - Optional hook configuration
 */
function afterEach(fn: () => void | Promise<void>, options?: { signal?: AbortSignal, timeout?: number }): void;
```

**Usage Examples:**

```javascript
import { describe, it, beforeEach, afterEach } from "test";

describe("Cache tests", () => {
  beforeEach(() => {
    // Initialize cache
    cache.init();
  });

  afterEach(() => {
    // Clear cache after each test
    cache.clear();
  });

  it("should store values", () => {
    cache.set("key", "value");
    if (cache.get("key") !== "value") {
      throw new Error("Cache store failed");
    }
  });

  it("should handle expiration", async () => {
    cache.set("key", "value", { ttl: 10 });
    await new Promise(resolve => setTimeout(resolve, 20));
    if (cache.get("key") !== null) {
      throw new Error("Cache expiration failed");
    }
  });
});

// Async afterEach
describe("Resource tests", () => {
  afterEach(async () => {
    await closeConnections();
    await cleanupTempFiles();
  });

  it("should manage resources", () => {
    // Test implementation
  });
});
```

## Hook Execution Order

Hooks execute in the following order for nested suites:

1. Outer `before` hooks
2. Inner `before` hooks
3. For each test:
   - Outer `beforeEach` hooks
   - Inner `beforeEach` hooks
   - **Test execution**
   - Inner `afterEach` hooks
   - Outer `afterEach` hooks
4. Inner `after` hooks
5. Outer `after` hooks

**Example:**

```javascript
import { describe, it, before, after, beforeEach, afterEach } from "test";

before(() => console.log("1. Global before"));
after(() => console.log("8. Global after"));

describe("Outer suite", () => {
  before(() => console.log("2. Outer before"));
  beforeEach(() => console.log("3. Outer beforeEach"));
  afterEach(() => console.log("6. Outer afterEach"));
  after(() => console.log("7. Outer after"));

  describe("Inner suite", () => {
    before(() => console.log("3. Inner before"));
    beforeEach(() => console.log("4. Inner beforeEach"));
    afterEach(() => console.log("5. Inner afterEach"));
    after(() => console.log("6. Inner after"));

    it("test case", () => {
      console.log("5. Test execution");
    });
  });
});
```

## Error Handling in Hooks

If a hook throws an error or returns a rejected promise:

- **before/beforeEach errors**: Skip the associated tests
- **after/afterEach errors**: Mark tests as failed but continue cleanup
- All hooks of the same type continue to run even if one fails

```javascript
describe("Error handling", () => {
  before(() => {
    throw new Error("Setup failed");
    // This will cause all tests in this suite to be skipped
  });

  afterEach(() => {
    // This runs even if the test or beforeEach failed
    cleanup();
  });

  it("this test will be skipped", () => {
    // Won't run due to before hook failure
  });
});
```

## Hook Scope

Hooks only apply to tests within their scope:

```javascript
// Global hooks - apply to all tests
before(() => {
  // Runs before any test
});

describe("Suite A", () => {
  // Suite-level hooks - only apply to tests in this suite
  beforeEach(() => {
    // Only runs before tests in Suite A
  });

  it("test 1", () => {});
  it("test 2", () => {});
});

describe("Suite B", () => {
  // Different suite-level hooks
  beforeEach(() => {
    // Only runs before tests in Suite B
  });

  it("test 3", () => {});
});
```

## Best Practices

1. **Keep hooks simple**: Focus on setup/cleanup, avoid complex logic
2. **Use async/await**: For asynchronous operations in hooks
3. **Clean up resources**: Always clean up in after/afterEach hooks
4. **Fail fast**: If setup fails, let the hook throw an error
5. **Scope appropriately**: Use the most specific hook scope possible

```javascript
describe("Best practices example", () => {
  let server;
  let client;

  before(async () => {
    // Expensive setup once per suite
    server = await startTestServer();
  });

  beforeEach(() => {
    // Fresh client for each test
    client = new ApiClient(server.url);
  });

  afterEach(() => {
    // Clean up test-specific resources
    client.close();
  });

  after(async () => {
    // Clean up suite-level resources
    await server.close();
  });

  it("should connect", async () => {
    await client.connect();
    if (!client.isConnected()) {
      throw new Error("Connection failed");
    }
  });
});
```