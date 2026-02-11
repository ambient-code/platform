"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import "@copilotkit/react-ui/styles.css";
import { CopilotKit } from "@copilotkit/react-core";
import { CopilotChat } from "@copilotkit/react-ui";
import { useCopilotChatInternal } from "@copilotkit/react-core";
import type { Message } from "@copilotkit/shared";
import { AlertCircle, X } from "lucide-react";
import { FeedbackModal, type FeedbackType } from "@/components/feedback/FeedbackModal";
import type { WorkflowMetadataResponse } from "@/services/api/workflows";
import {
  SessionActiveCtx,
  ResumeCtx,
  WorkflowMetadataCtx,
  PersistedFeedbackCtx,
  InputEnhancementsCtx,
  MessageTimestampsCtx,
  type ResumeCtxType,
  type WorkflowMetadataCtxType,
  type InputEnhancementsCtxType,
  type FeedbackState,
} from "./session-contexts";
import { DefaultToolRegistration } from "./ToolCallRender";
import { SessionAwareAssistantMessage } from "./SessionAwareAssistantMessage";
import { SessionAwareUserMessage } from "./SessionAwareUserMessage";
import { SessionAwareInput } from "./SessionAwareInput";

// ─── Public types ─────────────────────────────────────────────────────

type CopilotChatPanelProps = {
  projectName: string;
  sessionName: string;
  className?: string;
  isSessionActive?: boolean;
  workflowMetadata?: WorkflowMetadataResponse;
  onResume?: () => void;
  isResuming?: boolean;
  /** Render callback for the welcome experience, receives whether the chat has real messages */
  renderWelcome?: (hasMessages: boolean) => React.ReactNode;
  /** Input enhancement props — file upload, session queue, phase info */
  inputEnhancements?: InputEnhancementsCtxType;
};

// ─── Provider ─────────────────────────────────────────────────────────

/**
 * Provider component — wraps CopilotKit + tool nesting + default tool
 * registration.  Mount ONCE at the page level so that both desktop and
 * mobile layouts share a single agent connection (and a single threadId).
 *
 * `threadId` is pinned to `sessionName` so the identity survives a page
 * refresh.  CopilotKit will send the same threadId on every
 * `agent/connect`, which lets the backend proxy inject a persisted
 * MESSAGES_SNAPSHOT to restore the conversation.
 */
export function CopilotSessionProvider({
  projectName,
  sessionName,
  children,
}: {
  projectName: string;
  sessionName: string;
  children: React.ReactNode;
}) {
  const [mounted, setMounted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  React.useEffect(() => setMounted(true), []);

  const handleError = useCallback((err: unknown) => {
    console.error("[CopilotKit] Agent error:", err);
    const message =
      err instanceof Error
        ? err.message
        : typeof err === "string"
          ? err
          : "An unexpected error occurred";
    setError(message);
    setTimeout(() => setError(null), 15000);
  }, []);

  if (!mounted) {
    return null;
  }

  const runtimeUrl = `/api/copilotkit/${projectName}/${sessionName}`;

  return (
    <CopilotKit
      runtimeUrl={runtimeUrl}
      showDevConsole={process.env.NODE_ENV === "development"}
      agent={sessionName}
      threadId={sessionName}
      onError={handleError}
    >
      <DefaultToolRegistration />
      <ErrorBanner error={error} onDismiss={() => setError(null)} />
      {children}
    </CopilotKit>
  );
}

// ─── Error banner ─────────────────────────────────────────────────────

function ErrorBanner({ error, onDismiss }: { error: string | null; onDismiss: () => void }) {
  if (!error) return null;
  return (
    <div className="mx-3 mt-2 flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
      <div className="flex-1 whitespace-pre-wrap break-words font-mono text-xs">
        {error}
      </div>
      <button
        onClick={onDismiss}
        className="shrink-0 rounded p-0.5 hover:bg-destructive/20"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

// ─── Chat views ───────────────────────────────────────────────────────

/**
 * Chat view — renders the CopilotChat UI and feedback modal.
 * Mount inside a `CopilotSessionProvider`.  Safe to render in both
 * mobile and desktop layouts (only one is visible at a time via CSS).
 */
export function CopilotChatView({
  projectName,
  sessionName,
  className = "",
  isSessionActive = true,
  workflowMetadata,
  onResume,
  isResuming,
  renderWelcome,
  inputEnhancements,
}: CopilotChatPanelProps) {
  return (
    <ChatContent
      className={className}
      projectName={projectName}
      sessionName={sessionName}
      isSessionActive={isSessionActive}
      workflowMetadata={workflowMetadata}
      onResume={onResume}
      isResuming={isResuming}
      renderWelcome={renderWelcome}
      inputEnhancements={inputEnhancements}
    />
  );
}

/**
 * Combined component for backward compat — creates its own CopilotKit
 * context.  Prefer using `CopilotSessionProvider` + `CopilotChatView`
 * at the page level to avoid duplicate connections.
 */
export function CopilotChatPanel({
  projectName,
  sessionName,
  className = "",
  isSessionActive = true,
}: CopilotChatPanelProps) {
  return (
    <CopilotSessionProvider projectName={projectName} sessionName={sessionName}>
      <CopilotChatView
        projectName={projectName}
        sessionName={sessionName}
        className={className}
        isSessionActive={isSessionActive}
      />
    </CopilotSessionProvider>
  );
}

// ─── Message timestamp tracker ────────────────────────────────────────
//
// Watches the agent's message list and records the first-observed
// wall-clock time for each message ID.  Returns the map so the
// parent can provide it via MessageTimestampsCtx.

function useMessageTimestamps(): Record<string, number> {
  const { agent } = useCopilotChatInternal();
  const mapRef = useRef<Record<string, number>>({});
  const [, bump] = useState(0);

  useEffect(() => {
    if (!agent?.messages) return;
    let changed = false;
    for (const m of agent.messages) {
      if (m.id && !mapRef.current[m.id]) {
        mapRef.current[m.id] = Date.now();
        changed = true;
      }
    }
    if (changed) bump((n) => n + 1);
  }, [agent?.messages]);

  return mapRef.current;
}

// ─── Internal: chat content with feedback ─────────────────────────────

type PendingFeedback = {
  type: FeedbackType;
  messageId: string;
  messageContent: string;
  runId: string;
};

function ChatContent({
  className,
  projectName,
  sessionName,
  isSessionActive = true,
  workflowMetadata,
  onResume,
  isResuming,
  renderWelcome,
  inputEnhancements,
}: {
  className: string;
  projectName: string;
  sessionName: string;
  isSessionActive?: boolean;
  workflowMetadata?: WorkflowMetadataResponse;
  onResume?: () => void;
  isResuming?: boolean;
  renderWelcome?: (hasMessages: boolean) => React.ReactNode;
  inputEnhancements?: InputEnhancementsCtxType;
}) {
  const [pendingFeedback, setPendingFeedback] = useState<PendingFeedback | null>(null);
  const messageTimestamps = useMessageTimestamps();

  // ── Feedback map: derived from RAW events in the AG-UI stream ───
  // CopilotKit's internal messageFeedback (useState) resets on mount.
  // We persist feedback as RAW events (not CUSTOM) so they don't need
  // to be within run boundaries (RUN_STARTED/RUN_FINISHED).
  //
  // We subscribe to the AG-UI agent directly — the same mechanism the
  // CopilotKit web-inspector uses.  On reconnect the backend replays
  // the persisted RAW event, which fires our subscriber and populates
  // the map.
  const { agent } = useCopilotChatInternal();
  const [feedbackMap, setFeedbackMap] = useState<Record<string, FeedbackState>>({});


  useEffect(() => {
    if (!agent) return;
    const { unsubscribe } = agent.subscribe({
      onRawEvent: ({ event }) => {
        const inner = event.event as Record<string, unknown> | undefined;
        if (!inner || inner.name !== "ambient:feedback") return;
        const metaType = inner.metaType as string | undefined;
        const payload = inner.payload as Record<string, unknown> | undefined;
        const messageId = payload?.messageId as string | undefined;
        if (!messageId || !metaType) return;
        const state: FeedbackState = metaType === "thumbs_up" ? "thumbsUp" : "thumbsDown";
        setFeedbackMap((prev) => ({ ...prev, [messageId]: state }));
      },
    });
    return unsubscribe;
  }, [agent]);

  const workflowCtxValue = useMemo<WorkflowMetadataCtxType>(
    () => ({
      commands: workflowMetadata?.commands ?? [],
      agents: workflowMetadata?.agents ?? [],
    }),
    [workflowMetadata],
  );

  const extractContent = (msg: Message): string => {
    if (typeof msg.content === "string") return msg.content;
    if (Array.isArray(msg.content)) {
      return msg.content
        .filter((c): c is { type: "text"; text: string } => c.type === "text")
        .map((c) => c.text)
        .join("\n");
    }
    return "";
  };

  const handleThumbsUp = useCallback((message: Message) => {
    const runId = "runId" in message ? (message.runId as string) ?? "" : "";
    const messageId = message.id ?? "";
    if (messageId) setFeedbackMap((prev) => ({ ...prev, [messageId]: "thumbsUp" }));
    setPendingFeedback({
      type: "positive",
      messageId,
      messageContent: extractContent(message),
      runId,
    });
  }, []);

  const handleThumbsDown = useCallback((message: Message) => {
    const runId = "runId" in message ? (message.runId as string) ?? "" : "";
    const messageId = message.id ?? "";
    if (messageId) setFeedbackMap((prev) => ({ ...prev, [messageId]: "thumbsDown" }));
    setPendingFeedback({
      type: "negative",
      messageId,
      messageContent: extractContent(message),
      runId,
    });
  }, []);

  const handleFeedbackSubmitSuccess = useCallback(() => {
    setPendingFeedback(null);
  }, []);

  const resumeCtxValue = useMemo<ResumeCtxType>(
    () => ({ onResume, isResuming }),
    [onResume, isResuming],
  );
  const hasMessages = agent?.messages ? agent.messages.length > 0 : false;
  const showWelcome = !hasMessages;
  const welcomeNode = renderWelcome?.(hasMessages) ?? null;

  const defaultInputEnhancements = useMemo<InputEnhancementsCtxType>(
    () => ({ sessionPhase: "", queuedMessages: [], queuedCount: 0, isRunActive: false }),
    [],
  );
  const inputEnhancementsValue = inputEnhancements ?? defaultInputEnhancements;

  return (
    <div className={`flex flex-col h-full ${className}`}>
      <MessageTimestampsCtx.Provider value={messageTimestamps}>
      <PersistedFeedbackCtx.Provider value={feedbackMap}>
      <WorkflowMetadataCtx.Provider value={workflowCtxValue}>
      <SessionActiveCtx.Provider value={isSessionActive}>
      <ResumeCtx.Provider value={resumeCtxValue}>
      <InputEnhancementsCtx.Provider value={inputEnhancementsValue}>
        {/* Welcome experience — only shown when there are no messages yet */}
        {showWelcome && welcomeNode && (
          <div className="shrink-0 overflow-y-auto max-h-[60vh] px-3">
            {welcomeNode}
          </div>
        )}
        <CopilotChat
          className="h-full flex-1 min-h-0"
          AssistantMessage={SessionAwareAssistantMessage}
          UserMessage={SessionAwareUserMessage}
          Input={SessionAwareInput}
          labels={{
            initial: "",
            placeholder: "Send a message...",
          }}
          onThumbsUp={handleThumbsUp}
          onThumbsDown={handleThumbsDown}
        />
      </InputEnhancementsCtx.Provider>
      </ResumeCtx.Provider>
      </SessionActiveCtx.Provider>
      </WorkflowMetadataCtx.Provider>
      </PersistedFeedbackCtx.Provider>
      </MessageTimestampsCtx.Provider>

      {/* Feedback comment modal */}
      {pendingFeedback && (
        <FeedbackModal
          open={!!pendingFeedback}
          onOpenChange={(open) => {
            if (!open) setPendingFeedback(null);
          }}
          feedbackType={pendingFeedback.type}
          projectName={projectName}
          sessionName={sessionName}
          messageId={pendingFeedback.messageId}
          messageContent={pendingFeedback.messageContent}
          runId={pendingFeedback.runId}
          onSubmitSuccess={handleFeedbackSubmitSuccess}
        />
      )}
    </div>
  );
}
