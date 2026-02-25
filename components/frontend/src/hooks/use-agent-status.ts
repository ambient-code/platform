import { useMemo } from "react";
import type {
  AgenticSessionPhase,
  AgentStatus,
  MessageObject,
  ToolUseMessages,
} from "@/types/agentic-session";

function isAskUserQuestionTool(name: string): boolean {
  const normalized = name.toLowerCase().replace(/[^a-z]/g, "");
  return normalized === "askuserquestion";
}

/**
 * Derive agent status from session data and message stream.
 *
 * For the session detail page where the full message stream is available,
 * this provides accurate status including `waiting_input` detection.
 */
export function useAgentStatus(
  phase: AgenticSessionPhase | string,
  isRunActive: boolean,
  messages: Array<MessageObject | ToolUseMessages>,
): AgentStatus {
  return useMemo(() => {
    // Terminal states from session phase
    if (phase === "Completed") return "completed";
    if (phase === "Failed") return "failed";
    if (phase === "Stopped") return "idle";

    // Non-running phases
    if (phase !== "Running") return "idle";

    // Check if the last tool call is an unanswered AskUserQuestion
    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i];

      // Skip non-tool messages
      if (!("toolUseBlock" in msg)) continue;

      const toolMsg = msg as ToolUseMessages;
      if (isAskUserQuestionTool(toolMsg.toolUseBlock.name)) {
        // Check if it has a result (answered)
        const hasResult =
          toolMsg.resultBlock?.content !== undefined &&
          toolMsg.resultBlock?.content !== null &&
          toolMsg.resultBlock?.content !== "";
        if (!hasResult) {
          return "waiting_input";
        }
      }

      // Only check the most recent tool call
      break;
    }

    // Active processing
    if (isRunActive) return "working";

    // Running but idle between turns
    return "idle";
  }, [phase, isRunActive, messages]);
}

/**
 * Derive a simplified agent status from session phase alone.
 *
 * Used in the session list where per-session message streams are not available.
 */
export function deriveAgentStatusFromPhase(
  phase: AgenticSessionPhase | string,
): AgentStatus {
  switch (phase) {
    case "Running":
      return "working";
    case "Completed":
      return "completed";
    case "Failed":
      return "failed";
    case "Stopped":
      return "idle";
    default:
      return "idle";
  }
}
