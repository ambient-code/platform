import React from "react";
import type { WorkflowCommand, WorkflowAgent } from "@/services/api/workflows";
import type { QueuedMessageItem } from "@/hooks/use-session-queue";

// ─── Session-active context ───────────────────────────────────────────
//
// Lets custom AssistantMessage / Input components read session state
// without changing their component identity (avoids message list
// remounts when the value toggles).

export const SessionActiveCtx = React.createContext(true);

// ─── Persisted feedback context ──────────────────────────────────────
//
// CopilotKit stores messageFeedback as useState({}) — it resets on
// every mount.  The backend persists feedback as RAW events with
// name "ambient:feedback" in the AG-UI stream.  RAW events don't
// need run boundaries so they avoid AG-UI validation errors.
// On reconnect the backend replays the RAW event and we expose
// the derived feedback map via context so SessionAwareAssistantMessage
// can highlight thumbs buttons after a page refresh.

export type FeedbackState = "thumbsUp" | "thumbsDown";

export const PersistedFeedbackCtx = React.createContext<
  Record<string, FeedbackState>
>({});

// ─── Resume context ───────────────────────────────────────────────────
//
// When the session is hibernated (Stopped/Completed), the Input slot
// renders a resume banner instead of a disabled textarea.

export type ResumeCtxType = {
  onResume?: () => void;
  isResuming?: boolean;
};
export const ResumeCtx = React.createContext<ResumeCtxType>({});

// ─── Workflow metadata context ────────────────────────────────────────
//
// Bridges workflow commands and agents from the page-level data fetch
// down to the custom Input component (which only receives CopilotKit's
// fixed InputProps).  This enables slash-command and @agent autocomplete.

export type WorkflowMetadataCtxType = {
  commands: WorkflowCommand[];
  agents: WorkflowAgent[];
};

export const WorkflowMetadataCtx = React.createContext<WorkflowMetadataCtxType>({
  commands: [],
  agents: [],
});

// ─── Input enhancements context ──────────────────────────────────────
//
// Bridges page-level features (file upload, session queue, prompt
// history) down to the CopilotKit custom Input component.  CopilotKit
// only passes its fixed InputProps (onSend, onStop, inProgress) so
// anything extra flows through this context.

export type InputEnhancementsCtxType = {
  /** Upload a pasted/attached file to the session workspace */
  onPasteImage?: (file: File) => Promise<void>;
  /** Current session phase (Running, Creating, Completed, etc.) */
  sessionPhase: string;
  /** Queued messages (localStorage-backed, pre-Running and while agent busy) */
  queuedMessages: QueuedMessageItem[];
  /** Cancel a queued message by id */
  onCancelQueuedMessage?: (id: string) => void;
  /** Edit a queued message in-place */
  onUpdateQueuedMessage?: (id: string, content: string) => void;
  /** Clear all queued messages */
  onClearQueue?: () => void;
  /** Number of pending (unsent) queued messages */
  queuedCount: number;
  /** Whether the agent is actively processing (run in progress) */
  isRunActive: boolean;
  /** Mark a queued message as sent (called by QueueDrainer after CopilotKit sends it) */
  onMarkSent?: (id: string) => void;
  /** Add a message to the queue (for pre-Running or while agent is busy) */
  onQueueMessage?: (content: string) => void;
  /** Show/hide system messages toggle state */
  showSystemMessages?: boolean;
  /** Callback when system messages toggle changes */
  onShowSystemMessagesChange?: (show: boolean) => void;
  /** Callback when a slash command is clicked from the toolbar */
  onCommandClick?: (slashCommand: string) => void;
};

export const InputEnhancementsCtx = React.createContext<InputEnhancementsCtxType>({
  sessionPhase: "",
  queuedMessages: [],
  queuedCount: 0,
  isRunActive: false,
});

// ─── Message timestamps context ─────────────────────────────────────
//
// CopilotKit / AG-UI messages don't carry a `createdAt` field, so we
// record the wall-clock time each message ID is first observed by the
// UI and expose the map via context.  Custom message components read
// from this to render a timestamp badge.

export const MessageTimestampsCtx = React.createContext<
  Record<string, number>
>({});
