import type { ToolResultBlock } from "@/types/agentic-session";

export function normalizeToolName(name: string): string {
  return name.toLowerCase().replace(/[^a-z]/g, "");
}

export function isAskUserQuestionTool(name: string): boolean {
  return normalizeToolName(name) === "askuserquestion";
}

export function isExitPlanModeTool(name: string): boolean {
  return normalizeToolName(name) === "exitplanmode";
}

export function isHITLTool(name: string): boolean {
  const normalized = normalizeToolName(name);
  return normalized === "askuserquestion" || normalized === "exitplanmode";
}

export function hasToolResult(resultBlock?: ToolResultBlock): boolean {
  if (!resultBlock) return false;
  const content = resultBlock.content;
  if (!content) return false;
  if (typeof content === "string" && content.trim() === "") return false;
  return true;
}
