# Test Runner

Programmatic test runner for executing test files with advanced configuration options including concurrency, timeouts, and custom reporters. The runner provides fine-grained control over test execution and supports both individual file execution and batch processing.

## Capabilities

### Run Function

The main programmatic interface for running tests with configurable options.

```javascript { .api }
/**
 * Programmatic test runner with configurable options
 * @param options - Runner configuration options
 * @returns TestsStream for monitoring test progress and results
 */
function run(options?: RunOptions): TestsStream;

interface RunOptions {
  /** Number of concurrent test files to run. Default: 1 */
  concurrency?: number;
  /** Global timeout for all tests in milliseconds. Default: Infinity */
  timeout?: number;
  /** AbortSignal for cancelling test execution */
  signal?: AbortSignal;
  /** Array of test file paths to execute. Default: auto-discover */
  files?: string[];
  /** Inspector port for debugging test execution */
  inspectPort?: number;
}
```

**Usage Examples:**

```javascript
import { run } from "test";

// Basic usage - auto-discover and run tests
const stream = run();
stream.on('test:pass', (test) => {
  console.log(`✓ ${test.name}`);
});
stream.on('test:fail', (test) => {
  console.log(`✗ ${test.name}: ${test.error.message}`);
});

// Run specific files
run({
  files: ['./tests/unit/*.js', './tests/integration/*.js']
});

// Concurrent execution
run({
  concurrency: 4,
  timeout: 30000
});

// With abort signal
const controller = new AbortController();
run({
  signal: controller.signal,
  files: ['./long-running-test.js']
});

// Cancel after 10 seconds
setTimeout(() => controller.abort(), 10000);
```

### TestsStream

The runner returns a TestsStream that provides real-time test execution feedback and results.

```javascript { .api }
interface TestsStream extends EventEmitter {
  /** Stream of test events and results */
  on(event: 'test:start', listener: (test: TestInfo) => void): this;
  on(event: 'test:pass', listener: (test: TestInfo) => void): this;
  on(event: 'test:fail', listener: (test: TestInfo) => void): this;
  on(event: 'test:skip', listener: (test: TestInfo) => void): this;
  on(event: 'test:todo', listener: (test: TestInfo) => void): this;
  on(event: 'test:diagnostic', listener: (test: TestInfo, message: string) => void): this;
  on(event: 'end', listener: (results: TestResults) => void): this;
}

interface TestInfo {
  name: string;
  file: string;
  line?: number;
  column?: number;
  duration?: number;
  error?: Error;
}

interface TestResults {
  total: number;
  pass: number;
  fail: number;
  skip: number;
  todo: number;
  duration: number;
  files: string[];
}
```

**Usage Examples:**

```javascript
import { run } from "test";

const stream = run({
  files: ['./tests/**/*.test.js'],
  concurrency: 2
});

// Handle individual test events
stream.on('test:start', (test) => {
  console.log(`Starting ${test.name}...`);
});

stream.on('test:pass', (test) => {
  console.log(`✓ ${test.name} (${test.duration}ms)`);
});

stream.on('test:fail', (test) => {
  console.error(`✗ ${test.name}`);
  console.error(`  ${test.error.message}`);
  if (test.error.stack) {
    console.error(`  at ${test.file}:${test.line}:${test.column}`);
  }
});

stream.on('test:skip', (test) => {
  console.log(`- ${test.name} (skipped)`);
});

stream.on('test:todo', (test) => {
  console.log(`? ${test.name} (todo)`);
});

stream.on('test:diagnostic', (test, message) => {
  console.log(`# ${message}`);
});

// Handle completion
stream.on('end', (results) => {
  console.log(`\nResults:`);
  console.log(`  Total: ${results.total}`);
  console.log(`  Pass: ${results.pass}`);
  console.log(`  Fail: ${results.fail}`);
  console.log(`  Skip: ${results.skip}`);
  console.log(`  Todo: ${results.todo}`);
  console.log(`  Duration: ${results.duration}ms`);
  
  if (results.fail > 0) {
    process.exit(1);
  }
});
```

## File Discovery

When no files are specified, the runner automatically discovers test files using these patterns:

1. Files in `test/` directories with `.js`, `.mjs`, or `.cjs` extensions
2. Files ending with `.test.js`, `.test.mjs`, or `.test.cjs`
3. Files ending with `.spec.js`, `.spec.mjs`, or `.spec.cjs`
4. Excludes `node_modules` directories

**Examples:**

```javascript
// These files will be auto-discovered:
// test/user.js
// test/api/auth.test.js
// src/utils.spec.mjs
// integration.test.cjs

// These will be ignored:
// node_modules/package/test.js
// src/helper.js (no test pattern)
```

Manual file specification overrides auto-discovery:

```javascript
run({
  files: [
    './custom-tests/*.js',
    './e2e/**/*.test.js'
  ]
});
```

## Concurrency Control

The runner supports concurrent execution of test files to improve performance:

```javascript
// Sequential execution (default)
run({ concurrency: 1 });

// Parallel execution - 4 files at once
run({ concurrency: 4 });

// Unlimited concurrency
run({ concurrency: Infinity });

// Boolean concurrency (uses CPU count)
run({ concurrency: true });
```

**Important Notes:**
- Concurrency applies to test files, not individual tests within files
- Individual test concurrency is controlled by test options
- Higher concurrency may reveal race conditions in tests

## Timeout Configuration

Set global timeouts for test execution:

```javascript
// 30-second timeout for all tests
run({ timeout: 30000 });

// No timeout (default)
run({ timeout: Infinity });
```

Timeout behavior:
- Applies to individual test files, not the entire run
- Files that timeout are marked as failed
- Other files continue to execute

## Debugging Support

Enable debugging for test execution:

```javascript
// Run with Node.js inspector
run({
  inspectPort: 9229,
  files: ['./debug-this-test.js']
});

// Then connect with Chrome DevTools or VS Code
```

## Error Handling and Cancellation

```javascript
import { run } from "test";

const controller = new AbortController();
const stream = run({
  files: ['./tests/**/*.js'],
  signal: controller.signal
});

// Handle stream errors
stream.on('error', (error) => {
  console.error('Runner error:', error);
});

// Cancel execution
setTimeout(() => {
  console.log('Cancelling tests...');
  controller.abort();
}, 5000);

// Handle cancellation
controller.signal.addEventListener('abort', () => {
  console.log('Test execution cancelled');
});
```

## Integration Examples

### Custom Test Reporter

```javascript
import { run } from "test";
import fs from "fs";

const results = [];
const stream = run({ files: ['./tests/**/*.js'] });

stream.on('test:pass', (test) => {
  results.push({ status: 'pass', name: test.name, duration: test.duration });
});

stream.on('test:fail', (test) => {
  results.push({ 
    status: 'fail', 
    name: test.name, 
    error: test.error.message,
    duration: test.duration 
  });
});

stream.on('end', () => {
  // Write custom report
  fs.writeFileSync('test-results.json', JSON.stringify(results, null, 2));
});
```

### CI/CD Integration

```javascript
import { run } from "test";

async function runTests() {
  return new Promise((resolve, reject) => {
    const stream = run({
      files: process.env.TEST_FILES?.split(',') || undefined,
      concurrency: parseInt(process.env.TEST_CONCURRENCY) || 1,
      timeout: parseInt(process.env.TEST_TIMEOUT) || 30000
    });

    const results = { pass: 0, fail: 0, skip: 0, todo: 0 };

    stream.on('test:pass', () => results.pass++);
    stream.on('test:fail', () => results.fail++);
    stream.on('test:skip', () => results.skip++);
    stream.on('test:todo', () => results.todo++);

    stream.on('end', (finalResults) => {
      console.log(`Tests completed: ${finalResults.pass}/${finalResults.total} passed`);
      
      if (finalResults.fail > 0) {
        reject(new Error(`${finalResults.fail} tests failed`));
      } else {
        resolve(finalResults);
      }
    });

    stream.on('error', reject);
  });
}

// Usage in CI
runTests()
  .then(() => process.exit(0))
  .catch(() => process.exit(1));
```

### Watch Mode Implementation

```javascript
import { run } from "test";
import { watch } from "fs";

let currentRun = null;

function runTests() {
  if (currentRun) {
    currentRun.abort();
  }

  const controller = new AbortController();
  currentRun = controller;

  const stream = run({
    signal: controller.signal,
    files: ['./src/**/*.test.js']
  });

  stream.on('end', (results) => {
    console.log(`\nWatching for changes... (${results.pass}/${results.total} passed)`);
    currentRun = null;
  });

  stream.on('error', (error) => {
    if (error.name !== 'AbortError') {
      console.error('Run error:', error);
    }
    currentRun = null;
  });
}

// Initial run
runTests();

// Watch for file changes
watch('./src', { recursive: true }, (eventType, filename) => {
  if (filename.endsWith('.js')) {
    console.log(`\nFile changed: ${filename}`);
    runTests();
  }
});
```