import { useMemo } from "react";
import type {
  AgenticSessionPhase,
  AgentStatus,
} from "@/types/agentic-session";
import type { PlatformMessage } from "@/types/agui";

function isAskUserQuestionTool(name: string): boolean {
  const normalized = name.toLowerCase().replace(/[^a-z]/g, "");
  return normalized === "askuserquestion";
}

/**
 * Derive agent status from session data and the raw AG-UI message stream.
 *
 * For the session detail page where the full message stream is available,
 * this provides accurate status including `waiting_input` detection.
 */
export function useAgentStatus(
  phase: AgenticSessionPhase | string,
  isRunActive: boolean,
  messages: PlatformMessage[],
): AgentStatus {
  return useMemo(() => {
    // Terminal states from session phase
    if (phase === "Completed") return "completed";
    if (phase === "Failed") return "failed";
    if (phase === "Stopped") return "idle";

    // Non-running phases
    if (phase !== "Running") return "idle";

    // Scan backwards for the last tool call to check for unanswered AskUserQuestion.
    // Raw AG-UI messages store tool calls in msg.toolCalls[] (PlatformToolCall[]).
    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i];
      if (!msg.toolCalls || msg.toolCalls.length === 0) continue;

      // Check the last tool call on this message
      const lastTc = msg.toolCalls[msg.toolCalls.length - 1];
      if (lastTc.function?.name && isAskUserQuestionTool(lastTc.function.name)) {
        const hasResult =
          lastTc.result !== undefined &&
          lastTc.result !== null &&
          lastTc.result !== "";
        if (!hasResult) {
          return "waiting_input";
        }
      }

      // Only check the most recent message with tool calls
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
