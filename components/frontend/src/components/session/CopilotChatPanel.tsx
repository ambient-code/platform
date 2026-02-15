"use client";

import React, { useCallback, useMemo, useState } from "react";
import "@copilotkit/react-ui/styles.css";
import { CopilotKit } from "@copilotkit/react-core";
import { CopilotChat } from "@copilotkit/react-ui";
import type { Message } from "@copilotkit/shared";
import { AlertCircle, X } from "lucide-react";
import { FeedbackModal, type FeedbackType } from "@/components/feedback/FeedbackModal";
import type { WorkflowMetadataResponse } from "@/services/api/workflows";

import {
  SessionActiveCtx,
  ResumeCtx,
  WorkflowMetadataCtx,
  type ResumeCtxType,
  type WorkflowMetadataCtxType,
} from "./session-contexts";
import { DefaultToolRegistration } from "./ToolCallRender";
import { SessionAwareAssistantMessage } from "./SessionAwareAssistantMessage";
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
      showDevConsole={true}
      agent="session"
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
}: {
  className: string;
  projectName: string;
  sessionName: string;
  isSessionActive?: boolean;
  workflowMetadata?: WorkflowMetadataResponse;
  onResume?: () => void;
  isResuming?: boolean;
}) {
  const [pendingFeedback, setPendingFeedback] = useState<PendingFeedback | null>(null);

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
    setPendingFeedback({
      type: "positive",
      messageId: message.id ?? "",
      messageContent: extractContent(message),
      runId,
    });
  }, []);

  const handleThumbsDown = useCallback((message: Message) => {
    const runId = "runId" in message ? (message.runId as string) ?? "" : "";
    setPendingFeedback({
      type: "negative",
      messageId: message.id ?? "",
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

  return (
    <div className={`flex flex-col h-full ${className}`}>
      <WorkflowMetadataCtx.Provider value={workflowCtxValue}>
      <SessionActiveCtx.Provider value={isSessionActive}>
      <ResumeCtx.Provider value={resumeCtxValue}>
        <CopilotChat
          className="h-full flex-1"
          AssistantMessage={SessionAwareAssistantMessage}
          Input={SessionAwareInput}
          labels={{
            initial: "",
            placeholder: "Send a message...",
          }}
          onThumbsUp={handleThumbsUp}
          onThumbsDown={handleThumbsDown}
        />
      </ResumeCtx.Provider>
      </SessionActiveCtx.Provider>
      </WorkflowMetadataCtx.Provider>

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
