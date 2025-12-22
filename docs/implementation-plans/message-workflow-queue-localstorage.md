# Migration Plan: Message & Workflow Queuing to localStorage

## Overview
Move message and workflow queuing from in-memory React component state to browser localStorage with per-session identifiers. This ensures queued messages/workflows persist across page refreshes and are sent to the correct session when it becomes Running.

---

## Current Architecture

**Location:** `page.tsx` (Session Detail Page)

**Current State Variables:**
- `queuedMessages` - Array of message strings waiting to be sent
- `queuedMessagesSent` - Boolean tracking if queue has been processed
- `sentMessageCount` - Number of messages that were sent from queue
- `queuedWorkflow` - WorkflowConfig object waiting to be activated (in `use-workflow-management.ts`)

**Current Behavior:**
1. User sends message while session is not "Running" → message added to `queuedMessages`
2. Polling effect checks session phase every 2s
3. When phase becomes "Running" → messages sent sequentially via `aguiSendMessage`
4. Similar flow for workflow activation

**Limitations:**
- Queue lost on page refresh
- Queue lost on browser crash
- No persistence between navigation events
- No cross-tab synchronization

---

## New Architecture

**Storage Key Structure:**
```typescript
// Message queue key
`vteam:queue:messages:${projectName}:${sessionName}`

// Workflow queue key  
`vteam:queue:workflow:${projectName}:${sessionName}`

// Queue metadata key
`vteam:queue:meta:${projectName}:${sessionName}`
```

**Data Structures:**
```typescript
interface QueuedMessageItem {
  id: string;              // Unique ID for deduplication
  content: string;         // Message text
  timestamp: number;       // When queued
  sentAt?: number;         // When sent (if sent)
}

interface QueuedWorkflowItem {
  id: string;              // Workflow ID
  gitUrl: string;
  branch: string;
  path: string;
  timestamp: number;
  activatedAt?: number;    // When activated (if activated)
}

interface QueueMetadata {
  sessionPhase: string;    // Last known phase
  lastPolled: number;      // Last poll timestamp
  processing: boolean;     // Currently processing queue
}
```

---

## Implementation Steps

### Phase 1: Create localStorage Utility Hook

**File:** `src/hooks/use-session-queue.ts`

```typescript
export function useSessionQueue(projectName: string, sessionName: string) {
  // Load queue from localStorage on mount
  // Provide methods: addMessage, getMessages, markMessageSent, clearMessages
  // Provide methods: setWorkflow, getWorkflow, markWorkflowActivated, clearWorkflow
  // Provide methods: getMetadata, updateMetadata
  // Auto-cleanup: remove old entries (>24h) on load
}
```

**Key Features:**
- Reads/writes to localStorage with proper serialization
- Handles JSON parse errors gracefully
- Provides cleanup for stale entries
- Returns reactive state via useState
- Uses useEffect to persist on changes

### Phase 2: Update Session Page Component

**File:** `src/app/projects/[name]/sessions/[sessionName]/page.tsx`

**Changes:**
1. Replace state variables:
   ```typescript
   // OLD
   const [queuedMessages, setQueuedMessages] = useState<string[]>([]);
   const [queuedMessagesSent, setQueuedMessagesSent] = useState(false);
   
   // NEW
   const messageQueue = useSessionQueue(projectName, sessionName);
   ```

2. Update `sendChat` function:
   ```typescript
   // Instead of: setQueuedMessages(prev => [...prev, finalMessage])
   messageQueue.addMessage(finalMessage);
   ```

3. Update message processing effect (lines 316-339):
   ```typescript
   useEffect(() => {
     const phase = session?.status?.phase;
     const messages = messageQueue.getMessages();
     const unsentMessages = messages.filter(m => !m.sentAt);
     
     if (phase === "Running" && unsentMessages.length > 0) {
       const processMessages = async () => {
         for (const item of unsentMessages) {
           try {
             await aguiSendMessage(item.content);
             messageQueue.markMessageSent(item.id);
             await new Promise(resolve => setTimeout(resolve, 100));
           } catch (err) {
             errorToast("Failed to send queued message");
           }
         }
       };
       processMessages();
     }
   }, [session?.status?.phase, messageQueue]);
   ```

4. Update clear logic (lines 1035-1048):
   ```typescript
   // Instead of: setQueuedMessages([])
   messageQueue.clearMessages();
   ```

### Phase 3: Update Workflow Management Hook

**File:** `src/app/projects/[name]/sessions/[sessionName]/hooks/use-workflow-management.ts`

**Changes:**
1. Add localStorage integration:
   ```typescript
   const workflowQueue = useSessionQueue(projectName, sessionName);
   ```

2. Update `activateWorkflow` function (lines 32-85):
   ```typescript
   // Instead of: setQueuedWorkflow(workflow)
   workflowQueue.setWorkflow(workflow);
   ```

3. Update workflow processing effect in page.tsx (lines 282-289):
   ```typescript
   const queuedWorkflow = workflowQueue.getWorkflow();
   if (phase === "Running" && queuedWorkflow && !queuedWorkflow.activatedAt) {
     workflowManagement.activateWorkflow(queuedWorkflow, phase);
     workflowQueue.markWorkflowActivated(queuedWorkflow.id);
   }
   ```

### Phase 4: Update MessagesTab Component

**File:** `src/components/session/MessagesTab.tsx`

**Changes:**
1. Update props to accept queue data from localStorage:
   ```typescript
   queuedMessages?: QueuedMessageItem[];  // Array of objects, not strings
   ```

2. Update display logic (lines 310-324) to use new format:
   ```typescript
   queuedMessages.filter(m => !m.sentAt).map((item) => {
     const queuedUserMessage: MessageObject = {
       type: "user_message",
       content: { type: "text_block", text: item.content },
       timestamp: new Date(item.timestamp).toISOString(),
     };
     // ...
   })
   ```

### Phase 5: Add Cleanup & Migration

**File:** `src/hooks/use-session-queue.ts`

**Cleanup Logic:**
- On mount, check for entries older than 24 hours
- Remove stale entries to prevent localStorage bloat
- Max storage per session: 100 messages, 1 workflow

**Migration Logic:**
- Check for old in-memory state (dev warning only)
- Gracefully handle missing localStorage support (fallback to memory)

---

## Testing Plan

### Unit Tests
1. **localStorage utility:**
   - Test serialization/deserialization
   - Test cleanup of old entries
   - Test error handling (quota exceeded, parse errors)

2. **Queue operations:**
   - Test adding messages
   - Test marking messages as sent
   - Test clearing queue
   - Test workflow queue operations

### Integration Tests
1. **Session flow:**
   - Queue messages while session is Pending
   - Verify messages persist after refresh
   - Verify messages sent when session becomes Running
   - Verify queue cleared after processing

2. **Workflow flow:**
   - Queue workflow while session is Pending
   - Verify workflow persists after refresh
   - Verify workflow activated when session becomes Running

3. **Edge cases:**
   - Multiple tabs queuing messages
   - Browser quota exceeded
   - Corrupted localStorage data
   - Session deleted while messages queued

### Manual Testing Checklist
- [ ] Queue message, refresh page, verify message still queued
- [ ] Queue message, close tab, reopen, verify message still queued
- [ ] Queue workflow, refresh, verify workflow activates correctly
- [ ] Multiple messages queue correctly
- [ ] Queue clears after sending
- [ ] Old queues (24h+) are cleaned up
- [ ] Works across multiple sessions simultaneously
- [ ] localStorage quota handling

---

## Rollout Strategy

1. **Phase 1:** Implement utility hook with tests
2. **Phase 2:** Add to session page behind feature flag
3. **Phase 3:** Test in development/staging
4. **Phase 4:** Enable for beta users
5. **Phase 5:** Full rollout
6. **Phase 6:** Remove old in-memory implementation

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| localStorage quota exceeded | High | Implement max queue size, cleanup old entries, graceful fallback |
| localStorage disabled/blocked | Medium | Detect and fallback to in-memory queue with warning |
| Corrupted data in localStorage | Medium | Wrap all reads in try-catch, clear on parse error |
| Race conditions (multi-tab) | Low | Use timestamps to detect conflicts, prefer latest write |
| Browser compatibility | Low | localStorage widely supported, but add feature detection |

---

## Success Metrics

- [ ] Queued messages survive page refresh
- [ ] Queued messages sent successfully when session becomes Running
- [ ] No duplicate messages sent
- [ ] localStorage usage stays under 1MB per session
- [ ] Works in Chrome, Firefox, Safari, Edge
- [ ] Zero errors in production monitoring

---

## Future Enhancements

1. **Cross-tab sync:** Use `storage` event to sync queue across tabs
2. **Retry logic:** Auto-retry failed message sends
3. **Queue UI:** Show queued items with edit/remove capabilities
4. **Priority queue:** Support urgent messages
5. **Queue export:** Export queued items for debugging
6. **Analytics:** Track queue usage patterns

---

## Questions for Review

1. Should we implement cross-tab synchronization in v1?
2. What should max queue size be? (Proposed: 100 messages)
3. Should we show a warning when queue reaches certain size?
4. Should queued items expire after 24h or longer?
5. Do we need to encrypt sensitive data in localStorage?

