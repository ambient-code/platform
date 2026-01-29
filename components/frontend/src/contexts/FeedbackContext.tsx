"use client";

import React, { createContext, useContext, useMemo } from "react";
import type { MessageObject, ToolUseMessages } from "@/types/agentic-session";
import type { MessageFeedback } from "@/types/agui";

export type FeedbackContextValue = {
  projectName: string;
  sessionName: string;
  username: string;
  initialPrompt?: string;
  activeWorkflow?: string;
  messages: Array<MessageObject | ToolUseMessages>;
  // traceId from Langfuse if available (from session status)
  traceId?: string;
  // Track which messages have received feedback (messageId -> feedback type)
  messageFeedback?: Map<string, MessageFeedback>;
};

const FeedbackContext = createContext<FeedbackContextValue | null>(null);

type FeedbackProviderProps = {
  projectName: string;
  sessionName: string;
  username: string;
  initialPrompt?: string;
  activeWorkflow?: string;
  messages: Array<MessageObject | ToolUseMessages>;
  traceId?: string;
  messageFeedback?: Map<string, MessageFeedback>;
  children: React.ReactNode;
};

export function FeedbackProvider({
  projectName,
  sessionName,
  username,
  initialPrompt,
  activeWorkflow,
  messages,
  traceId,
  messageFeedback,
  children,
}: FeedbackProviderProps) {
  const value = useMemo(
    () => ({
      projectName,
      sessionName,
      username,
      initialPrompt,
      activeWorkflow,
      messages,
      traceId,
      messageFeedback,
    }),
    [projectName, sessionName, username, initialPrompt, activeWorkflow, messages, traceId, messageFeedback]
  );

  return (
    <FeedbackContext.Provider value={value}>
      {children}
    </FeedbackContext.Provider>
  );
}

export function useFeedbackContext() {
  const context = useContext(FeedbackContext);
  if (!context) {
    throw new Error("useFeedbackContext must be used within a FeedbackProvider");
  }
  return context;
}

// Optional hook that doesn't throw if context is missing
export function useFeedbackContextOptional() {
  return useContext(FeedbackContext);
}
