import React from "react";
import type { WorkflowCommand, WorkflowAgent } from "@/services/api/workflows";

// ─── Session-active context ───────────────────────────────────────────
//
// Lets custom AssistantMessage / Input components read session state
// without changing their component identity (avoids message list
// remounts when the value toggles).

export const SessionActiveCtx = React.createContext(true);

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
