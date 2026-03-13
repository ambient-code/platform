# React State Stability & Rendering Patterns

Critical patterns for ensuring stable, predictable values in React components and avoiding common rendering anti-patterns.

## Core Principle

**Values displayed in the UI should be stable and not recalculated on every render** unless they explicitly need to be reactive to specific dependencies.

## Common Anti-Patterns

### ❌ Anti-Pattern 1: Recalculating Time on Every Render

**The Bug:**
```tsx
// ❌ BAD: This will show the CURRENT time on every render
function MessageItem({ message }: { message: Message }) {
  return (
    <div>
      <span>{message.content}</span>
      <time>{new Date().toLocaleTimeString()}</time>  {/* WRONG! */}
    </div>
  )
}
```

**Why it's wrong:**
- `new Date()` creates a new timestamp every time the component renders
- If parent re-renders or state changes, all messages show the same "current" time
- By the end of a conversation, all timestamps will appear identical

**Symptom:**
- Timestamps that update dynamically to show the current time
- All messages eventually showing the same timestamp
- Time values that "drift" as you interact with the page

**✅ Fix: Use Stable Message Data**
```tsx
// ✅ GOOD: Display the timestamp from the message data
function MessageItem({ message }: { message: Message }) {
  return (
    <div>
      <span>{message.content}</span>
      <time>{new Date(message.timestamp).toLocaleTimeString()}</time>
    </div>
  )
}
```

**✅ Alternative: Memoize Computed Values**
```tsx
// ✅ GOOD: Memoize if you must compute
function MessageItem({ message }: { message: Message }) {
  const formattedTime = useMemo(
    () => new Date(message.timestamp).toLocaleTimeString(),
    [message.timestamp]
  )

  return (
    <div>
      <span>{message.content}</span>
      <time>{formattedTime}</time>
    </div>
  )
}
```

### ❌ Anti-Pattern 2: Generating IDs on Every Render

```tsx
// ❌ BAD: New ID on every render
function FormField() {
  const id = `field-${Math.random()}`  // WRONG!
  return <input id={id} />
}

// ✅ GOOD: Stable ID using useId
function FormField() {
  const id = useId()
  return <input id={id} />
}

// ✅ GOOD: ID from props or stable source
function FormField({ fieldName }: { fieldName: string }) {
  const id = `field-${fieldName}`
  return <input id={id} />
}
```

### ❌ Anti-Pattern 3: Recreating Objects/Arrays on Render

```tsx
// ❌ BAD: New array reference on every render
function UserList() {
  const emptyState = { message: "No users" }  // New object every render!
  const users = useUsers()

  if (!users.length) return <div>{emptyState.message}</div>
}

// ✅ GOOD: Define outside component or use useMemo
const EMPTY_STATE = { message: "No users" }

function UserList() {
  const users = useUsers()
  if (!users.length) return <div>{EMPTY_STATE.message}</div>
}

// ✅ GOOD: Use useMemo for computed objects
function UserList() {
  const users = useUsers()
  const emptyState = useMemo(
    () => ({ message: `No users in ${organizationName}` }),
    [organizationName]
  )

  if (!users.length) return <div>{emptyState.message}</div>
}
```

## Debugging Timestamp/Time-Related Issues

### Investigation Checklist

When investigating timestamp bugs, check:

1. **Where is the time value coming from?**
   - [ ] From server/API data (message.timestamp)?
   - [ ] From local state (useState)?
   - [ ] Computed on every render (new Date())?  ← **Most likely culprit**

2. **Is the value being recalculated?**
   - [ ] Is `new Date()` called without arguments?
   - [ ] Is `Date.now()` called in the render?
   - [ ] Are time formatting functions called with no stable input?

3. **Is the value memoized?**
   - [ ] Is `useMemo` used for expensive computations?
   - [ ] Are dependencies specified correctly?
   - [ ] Could this value be computed once at data fetch time?

### Diagnostic Pattern

```tsx
// Add this to suspect components to track re-renders
function MessageItem({ message }: { message: Message }) {
  console.log('MessageItem rendered at:', new Date().toISOString())
  console.log('Message timestamp:', message.timestamp)

  // If these logs show different times but same message.timestamp,
  // it means the component is re-rendering but data is stable (good!)

  // If message.timestamp is undefined or changes unexpectedly,
  // that's your data problem

  return <div>...</div>
}
```

### Common Root Causes

**Timestamps showing current time instead of message time:**
- ✓ Using `new Date()` without an argument in the render
- ✓ Using `Date.now()` in the render
- ✓ Formatting function called on each render without memoization

**Timestamps changing unexpectedly:**
- ✓ Parent component passing new `Date()` as prop
- ✓ State being updated with current time on each render
- ✓ Time formatting happening in wrong lifecycle stage

**All timestamps showing the same value:**
- ✓ `new Date()` being called during render (most common!)
- ✓ Single timestamp being reused across all items
- ✓ Timestamp not being included in API response

## Best Practices

### 1. Store Timestamps as ISO Strings or Unix Time

```tsx
// ✅ GOOD: Store as ISO string from server
type Message = {
  id: string
  content: string
  timestamp: string  // "2024-01-15T10:30:00Z"
}

// Format for display
function formatTimestamp(isoString: string): string {
  return new Date(isoString).toLocaleTimeString()
}
```

### 2. Format Timestamps at the Data Layer (Ideal)

```tsx
// ✅ BEST: Pre-format in the API response or query
type Message = {
  id: string
  content: string
  timestamp: string
  formattedTimestamp: string  // Already formatted
}

// In your API client or React Query select:
const { data } = useQuery({
  queryKey: ['messages'],
  queryFn: fetchMessages,
  select: (messages) => messages.map(msg => ({
    ...msg,
    formattedTimestamp: new Date(msg.timestamp).toLocaleTimeString()
  }))
})
```

### 3. Use React.memo for Timestamp Display Components

```tsx
// ✅ GOOD: Prevent unnecessary re-renders
const Timestamp = React.memo(({ timestamp }: { timestamp: string }) => {
  const formatted = useMemo(
    () => new Date(timestamp).toLocaleTimeString(),
    [timestamp]
  )

  return <time dateTime={timestamp}>{formatted}</time>
})
```

### 4. Freeze Computed Values When Created

```tsx
// ✅ GOOD: Compute once when message arrives
function useMessages() {
  return useQuery({
    queryKey: ['messages'],
    queryFn: fetchMessages,
    select: (data) => data.messages.map(msg => ({
      ...msg,
      // Freeze the display time when message first arrives
      displayTime: new Date(msg.timestamp).toLocaleTimeString()
    }))
  })
}
```

## Quick Reference

| Scenario | ❌ Anti-Pattern | ✅ Pattern |
|----------|----------------|-----------|
| Display message time | `new Date().toLocaleTimeString()` | `new Date(message.timestamp).toLocaleTimeString()` |
| Display current time | Inline `new Date()` | `useState` + `useEffect` interval |
| Format timestamp | In render without memo | `useMemo` or format in data layer |
| Generate ID | `Math.random()` in render | `useId()` or stable prop |
| Create object | Inline object literal | Constant outside component or `useMemo` |

## When You See a Timestamp Bug

**First, verify it's a rendering issue, not a data issue:**

1. Check the raw data: `console.log(message.timestamp)`
2. If timestamp data is correct but display is wrong → **Rendering issue**
3. If timestamp data is undefined/changing → **Data fetching issue**

**For rendering issues:**
- Search for `new Date()` without arguments in component
- Search for `Date.now()` in component
- Check if time formatting is inside render without memoization

**For data issues:**
- Check API response structure
- Verify timestamp is included in GraphQL/REST query
- Check if backend is setting timestamps correctly

## Summary

**Golden Rule:** Never compute time-sensitive values (timestamps, IDs, random numbers) directly in the render phase unless you explicitly want them to change on every render.

**Default approach:**
1. Store stable values from data source (API, props, state)
2. Memoize any transformations with `useMemo`
3. Use stable value generators (`useId`, `useState` with initializer)
4. When in doubt, add a console.log to verify value stability
