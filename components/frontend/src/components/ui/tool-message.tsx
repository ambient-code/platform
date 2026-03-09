"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ToolResultBlock, ToolUseBlock, ToolUseMessages } from "@/types/agentic-session";
import {
  ChevronDown,
  ChevronRight,
  Loader2,
  Check,
  X,
  Cog,
} from "lucide-react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { formatTimestamp } from "@/lib/format-timestamp";

export type ToolMessageProps = {
  toolUseBlock?: ToolUseBlock;
  resultBlock?: ToolResultBlock;
  childToolCalls?: ToolUseMessages[];
  className?: string;
  borderless?: boolean;
  timestamp?: string;
};

const formatToolName = (toolName?: string) => {
  if (!toolName) return "Unknown Tool";
  return toolName
    .replace(/^mcp__/, "")
    .replace(/_/g, " ")
    .split(" ")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};

const formatToolInput = (input?: string) => {
  if (!input) return "{}";
  try {
    const parsed = JSON.parse(input);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return input;
  }
};

type ExpandableMarkdownProps = {
  content: string;
  maxLength?: number;
  className?: string;
};

const ExpandableMarkdown: React.FC<ExpandableMarkdownProps> = ({
  content,
  maxLength = 2000,
  className,
}) => {
  const [expanded, setExpanded] = useState(false);
  const shouldTruncate = content.length > maxLength;
  const display = expanded || !shouldTruncate ? content : content.substring(0, maxLength);

  const markdownComponents: Components = {
    code: ({
      inline,
      className,
      children,
      ...props
    }: {
      inline?: boolean;
      className?: string;
      children?: React.ReactNode;
    } & React.HTMLAttributes<HTMLElement>) => {
      return inline ? (
        <code className="bg-code-bg text-code-foreground px-1 py-0.5 rounded text-[0.9em]" {...(props as React.HTMLAttributes<HTMLElement>)}>
          {children}
        </code>
      ) : (
        <pre className="bg-code-bg text-code-foreground p-3 rounded-lg text-sm overflow-x-auto border border-border/50">
          <code className={className} {...(props as React.HTMLAttributes<HTMLElement>)}>
            {children}
          </code>
        </pre>
      );
    },
    p: ({ children }) => <div className="text-foreground/75 leading-relaxed mb-2 text-base">{children}</div>,
    h1: ({ children }) => <h1 className="text-xl font-bold text-foreground mb-2">{children}</h1>,
    h2: ({ children }) => <h2 className="text-lg font-semibold text-foreground mb-2">{children}</h2>,
    h3: ({ children }) => <h3 className="text-base font-medium text-foreground mb-1">{children}</h3>,
  };

  return (
    <div className={cn("max-w-none", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {display}
      </ReactMarkdown>
      {shouldTruncate && (
        <div className="mt-2">
          <button
            type="button"
            onClick={() => setExpanded(!expanded)}
            className="text-sm px-2.5 py-1 rounded-md border bg-card hover:bg-muted/50 text-foreground/80 transition-colors"
          >
            {expanded ? "Show less" : "Show more"}
          </button>
        </div>
      )}
    </div>
  );
};

// Helpers for Subagent rendering
const getInitials = (name?: string) => {
  if (!name) return "?";
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
};

const hashStringToNumber = (str: string) => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash);
};

/*
 * Refined agent color palette — muted, cohesive tones derived from the
 * three brand accents (blue 255, purple 290, red 25) plus complementary hues.
 * Uses semantic design tokens for consistency across themes.
 */
const getColorClassesForName = (name: string) => {
  const colorChoices = [
    // Blue family (primary)
    { avatarBg: "bg-primary", badgeBg: "bg-primary", cardBg: "bg-primary/5 dark:bg-primary/8", border: "border-primary/20 dark:border-primary/25", badgeText: "text-primary dark:text-primary", badgeBorder: "border-primary/20 dark:border-primary/25" },
    // Purple family (chart-2)
    { avatarBg: "bg-chart-2", badgeBg: "bg-chart-2", cardBg: "bg-chart-2/5 dark:bg-chart-2/8", border: "border-chart-2/20 dark:border-chart-2/25", badgeText: "text-chart-2 dark:text-chart-2", badgeBorder: "border-chart-2/20 dark:border-chart-2/25" },
    // Teal family (chart-4)
    { avatarBg: "bg-chart-4", badgeBg: "bg-chart-4", cardBg: "bg-chart-4/5 dark:bg-chart-4/8", border: "border-chart-4/20 dark:border-chart-4/25", badgeText: "text-chart-4 dark:text-chart-4", badgeBorder: "border-chart-4/20 dark:border-chart-4/25" },
    // Green family (chart-5)
    { avatarBg: "bg-chart-5", badgeBg: "bg-chart-5", cardBg: "bg-chart-5/5 dark:bg-chart-5/8", border: "border-chart-5/20 dark:border-chart-5/25", badgeText: "text-chart-5 dark:text-chart-5", badgeBorder: "border-chart-5/20 dark:border-chart-5/25" },
    // Warm accent (destructive / red family)
    { avatarBg: "bg-destructive", badgeBg: "bg-destructive", cardBg: "bg-destructive/5 dark:bg-destructive/8", border: "border-destructive/20 dark:border-destructive/25", badgeText: "text-destructive dark:text-destructive", badgeBorder: "border-destructive/20 dark:border-destructive/25" },
    // Muted blue-gray
    { avatarBg: "bg-muted-foreground", badgeBg: "bg-muted-foreground", cardBg: "bg-muted/50 dark:bg-muted/30", border: "border-border dark:border-border", badgeText: "text-muted-foreground dark:text-muted-foreground", badgeBorder: "border-border dark:border-border" },
  ];
  const idx = hashStringToNumber(name) % colorChoices.length;
  return colorChoices[idx];
};

// Helper to convert Python literal to JSON-parseable string
const pythonLiteralToJson = (pythonStr: string): string => {
  let result = '';
  let inString = false;
  let stringChar = '';
  let escaped = false;

  for (let i = 0; i < pythonStr.length; i++) {
    const char = pythonStr[i];

    if (escaped) {
      if (inString) {
        if (char === "'") {
          result += "'";
        } else if (char === '"') {
          result += '\\"';
        } else {
          result += '\\' + char;
        }
      } else {
        result += '\\' + char;
      }
      escaped = false;
      continue;
    }

    if (char === '\\') {
      escaped = true;
      continue;
    }

    if (char === "'" || char === '"') {
      if (!inString) {
        inString = true;
        stringChar = char;
        result += '"';
      } else if (char === stringChar) {
        inString = false;
        stringChar = '';
        result += '"';
      } else {
        if (char === '"') {
          result += '\\"';
        } else {
          result += "'";
        }
      }
      continue;
    }

    if (inString) {
      result += char;
      continue;
    }

    if (!inString) {
      if (pythonStr.substr(i, 4) === 'True') {
        result += 'true';
        i += 3;
        continue;
      }
      if (pythonStr.substr(i, 5) === 'False') {
        result += 'false';
        i += 4;
        continue;
      }
      if (pythonStr.substr(i, 4) === 'None') {
        result += 'null';
        i += 3;
        continue;
      }
    }

    result += char;
  }

  return result;
};

const extractTextFromResultContent = (content: unknown): string => {
  try {
    if (typeof content === "string") {
      if (content.trim().startsWith("[") || content.trim().startsWith("{")) {
        try {
          const parsed = JSON.parse(content);
          return extractTextFromResultContent(parsed);
        } catch {
          try {
            const jsonStr = pythonLiteralToJson(content);
            const parsed = JSON.parse(jsonStr);
            return extractTextFromResultContent(parsed);
          } catch {
            console.warn('Failed to parse result content, showing raw text');
            return content;
          }
        }
      }
      return content;
    }

    if (Array.isArray(content)) {
      const texts = content
        .map((item) => {
          if (item && typeof item === "object" && "text" in (item as Record<string, unknown>)) {
            return String((item as Record<string, unknown>).text ?? "");
          }
          return "";
        })
        .filter(Boolean);
      if (texts.length) return texts.join("\n\n");
    }

    if (content && typeof content === "object") {
      const maybe = (content as Record<string, unknown>).content;
      if (Array.isArray(maybe)) {
        const texts = maybe
          .map((item) => {
            if (item && typeof item === "object" && "text" in (item as Record<string, unknown>)) {
              return String((item as Record<string, unknown>).text ?? "");
            }
            return "";
          })
          .filter(Boolean);
        if (texts.length) return texts.join("\n\n");
      }
    }

    return JSON.stringify(content ?? "");
  } catch {
    return String(content ?? "");
  }
};

// Generate smart summary for tool calls based on tool name and input
const generateToolSummary = (toolName: string, input?: Record<string, unknown>): string => {
  if (!input || Object.keys(input).length === 0) return formatToolName(toolName);

  if (toolName.toLowerCase().includes("websearch") || toolName.toLowerCase().includes("web_search")) {
    const query = input.query as string | undefined;
    if (query) return `Searching the web for "${query}"`;
  }

  if (toolName.toLowerCase().includes("read") && (input.file || input.path || input.target_file)) {
    const file = (input.file || input.path || input.target_file) as string;
    return `Reading ${file}`;
  }

  if (toolName.toLowerCase().includes("write") && (input.file || input.path || input.target_file)) {
    const file = (input.file || input.path || input.target_file) as string;
    return `Writing to ${file}`;
  }

  if (toolName.toLowerCase().includes("grep") || toolName.toLowerCase().includes("search")) {
    const pattern = input.pattern as string | undefined;
    const path = input.path as string | undefined;
    if (pattern && path) return `Searching for "${pattern}" in ${path}`;
    if (pattern) return `Searching for "${pattern}"`;
  }

  if (toolName.toLowerCase().includes("command") || toolName.toLowerCase().includes("terminal")) {
    const command = input.command as string | undefined;
    if (command) {
      const truncated = command.length > 50 ? command.substring(0, 50) + "..." : command;
      return `Running: ${truncated}`;
    }
  }

  const firstStringValue = Object.values(input).find(v => typeof v === 'string' && v.length > 0) as string | undefined;
  if (firstStringValue) {
    const truncated = firstStringValue.length > 60 ? firstStringValue.substring(0, 60) + "..." : firstStringValue;
    return truncated;
  }

  return formatToolName(toolName);
};

// Child Tool Call component for hierarchical rendering (collapsed by default)
type ChildToolCallProps = {
  toolUseBlock?: ToolUseBlock;
  resultBlock?: ToolResultBlock;
};

const ChildToolCall: React.FC<ChildToolCallProps> = ({ toolUseBlock, resultBlock }) => {
  const [expanded, setExpanded] = useState(false);

  const hasActualResult = Boolean(
    resultBlock &&
    resultBlock.content !== undefined &&
    resultBlock.content !== null &&
    (() => {
      const content = resultBlock.content;
      if (content === "") return false;
      if (Array.isArray(content) && content.length === 0) return false;
      if (typeof content === 'object' && !Array.isArray(content) && Object.keys(content).length === 0) return false;
      if (typeof content === 'string' && content.trim() === '') return false;
      if (typeof content === 'string' && (content === '""' || content === "''")) return false;
      return true;
    })()
  );

  const isError = resultBlock?.is_error === true;
  const isSuccess = hasActualResult && !isError;
  const isPending = !hasActualResult && !isError;

  const toolName = toolUseBlock?.name || "unknown_tool";

  let toolInput: Record<string, unknown> | undefined;
  if (toolUseBlock?.input) {
    if (typeof toolUseBlock.input === 'string') {
      try {
        toolInput = JSON.parse(toolUseBlock.input) as Record<string, unknown>;
      } catch {
        toolInput = { value: toolUseBlock.input };
      }
    } else {
      toolInput = toolUseBlock.input as Record<string, unknown>;
    }
  }

  const collapsedSummary = generateToolSummary(toolName, toolInput);

  return (
    <div className="py-0.5">
      <div
        className="flex items-center gap-2 cursor-pointer hover:bg-muted/40 rounded-md px-2 py-1.5 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {isError && <X className="w-3.5 h-3.5 text-destructive flex-shrink-0" />}
        {isSuccess && <Check className="w-3.5 h-3.5 text-chart-5 flex-shrink-0" />}
        {isPending && <Loader2 className="w-3.5 h-3.5 animate-spin text-primary flex-shrink-0" />}

        <Badge variant="secondary" className="text-sm px-2 py-0.5 flex-shrink-0">
          {formatToolName(toolName)}
        </Badge>

        {!expanded && (
          <span className="text-sm text-muted-foreground truncate flex-1">
            {collapsedSummary}
          </span>
        )}

        <ChevronRight className={cn(
          "w-3.5 h-3.5 text-muted-foreground transition-transform flex-shrink-0",
          expanded && "rotate-90"
        )} />
      </div>

      {expanded && (
        <div className="mt-1 ml-5 space-y-2 bg-muted/20 rounded-lg p-3 border border-border/50">
          {toolInput && Object.keys(toolInput).length > 0 && (
            <div>
              <div className="font-medium text-foreground/60 mb-1 text-sm">Input</div>
              <pre className="text-sm overflow-x-auto text-muted-foreground bg-code-bg rounded-md p-2">
                {JSON.stringify(toolInput, null, 2)}
              </pre>
            </div>
          )}
          {resultBlock?.content && (
            <div>
              <div className="font-medium text-foreground/60 mb-1 text-sm">
                Result {isError && <span className="text-destructive">(Error)</span>}
              </div>
              <ExpandableMarkdown
                className="prose-sm"
                content={extractTextFromResultContent(resultBlock.content as unknown)}
                maxLength={500}
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export const ToolMessage = React.forwardRef<HTMLDivElement, ToolMessageProps>(
  ({ toolUseBlock, resultBlock, childToolCalls, className, borderless, timestamp, ...props }, ref) => {
    const [isExpanded, setIsExpanded] = useState(false);

    const toolResultBlock = resultBlock;

    const hasActualResult = Boolean(
      toolResultBlock &&
      toolResultBlock.content !== undefined &&
      toolResultBlock.content !== null &&
      (() => {
        const content = toolResultBlock.content;
        if (content === "") return false;
        if (Array.isArray(content) && content.length === 0) return false;
        if (typeof content === 'object' && !Array.isArray(content) && Object.keys(content).length === 0) return false;
        if (typeof content === 'string' && content.trim() === '') return false;
        if (typeof content === 'string' && (content === '""' || content === "''")) return false;
        return true;
      })()
    );

    const isToolCall = Boolean(toolUseBlock && !hasActualResult);
    const isToolResult = hasActualResult;

    const toolName = formatToolName(toolUseBlock?.name);
    const isError = toolResultBlock?.is_error === true;
    const isLoading = isToolCall && !isError;
    const isSuccess = isToolResult && !isError;

    // Subagent detection and data
    const inputData = (toolUseBlock?.input ?? undefined) as unknown as Record<string, unknown> | undefined;
    const subagentType = (inputData?.subagent_type as string) || undefined;
    const subagentDescription = (inputData?.description as string) || undefined;
    const subagentPrompt = (inputData?.prompt as string) || undefined;
    const isSubagent = Boolean(subagentType);
    const subagentClasses = subagentType ? getColorClassesForName(subagentType) : undefined;
    const displayName = isSubagent ? subagentType : toolName;

    const isCompact = !isSubagent;

    const formattedTime = formatTimestamp(timestamp);

    return (
      <div ref={ref} className={cn(isCompact ? "mb-1" : "mb-4", className)} {...props}>
        <div className="flex items-start space-x-3">
          {/* Avatar */}
          <div className="flex-shrink-0">
            {isSubagent ? (
              <div className={cn("w-8 h-8 rounded-full flex items-center justify-center", subagentClasses?.avatarBg)}>
                <span className="text-white text-sm font-semibold">
                  {getInitials(subagentType)}
                </span>
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full flex items-center justify-center bg-muted-foreground/60">
                <Cog className="w-4 h-4 text-background" />
              </div>
            )}
          </div>

          {/* Tool Message Content */}
          <div className="flex-1 min-w-0">
            {/* Timestamp */}
            {formattedTime && (
              <div className="text-sm text-muted-foreground/60 mb-1">
                {formattedTime}
              </div>
            )}
            <div
              className={cn(
                isCompact ? "" : (borderless ? "p-0" : "rounded-lg border shadow-sm"),
                isSubagent ? subagentClasses?.cardBg : "",
                isSubagent ? subagentClasses?.border : undefined
              )}
            >
              {/* Collapsible Header */}
              <div
                className={cn(
                  "flex items-center justify-between cursor-pointer hover:bg-muted/40 transition-colors rounded-md",
                  isCompact ? "py-1.5 px-0" : "p-3"
                )}
                onClick={() => setIsExpanded(!isExpanded)}
              >
                <div className={cn("flex items-center flex-1 min-w-0", isCompact ? "space-x-2" : "space-x-2.5")}>
                  {/* Status Icon */}
                  <div className="flex-shrink-0">
                    {isLoading && (
                      <Loader2 className={cn(isCompact ? "w-3.5 h-3.5" : "w-4 h-4", "text-primary animate-spin")} />
                    )}
                    {isSuccess && <Check className={cn(isCompact ? "w-3.5 h-3.5" : "w-4 h-4", "text-chart-5")} />}
                    {isError && <X className={cn(isCompact ? "w-3.5 h-3.5" : "w-4 h-4", "text-destructive")} />}
                  </div>

                  {/* Tool Name Badge */}
                  <div className="flex-shrink-0">
                    <Badge
                      className={cn(
                        "text-sm text-white",
                        isLoading && "bg-primary",
                        isError && "bg-destructive",
                        isSuccess && "bg-chart-5",
                        isSubagent && subagentClasses?.badgeBg,
                        isCompact && "!py-0.5 px-2 leading-tight"
                      )}
                    >
                      {displayName}
                    </Badge>
                  </div>

                  {/* Title/Description */}
                  <div className="flex-1 min-w-0 text-base text-muted-foreground truncate">
                    {isSubagent ? (
                      <span className="truncate">
                        {subagentDescription || subagentPrompt || "Working..."}
                      </span>
                    ) : (
                      <span className="truncate">
                        {generateToolSummary(toolUseBlock?.name || "", inputData)}
                      </span>
                    )}
                  </div>

                  {/* Expand/Collapse Icon */}
                  <div className="flex-shrink-0">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-muted-foreground/60" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-muted-foreground/60" />
                    )}
                  </div>
                </div>
              </div>

              {/* Subagent primary content — Input → Activity → Result */}
              {isSubagent && isExpanded ? (
                <div className="px-3 pb-3 space-y-3">
                  {/* 1. INPUT */}
                  {subagentPrompt && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-foreground/50 uppercase tracking-wide">
                        Prompt
                      </h4>
                      <div className="rounded-lg p-3 overflow-x-auto bg-muted/20 border border-border/50 text-base text-foreground/75">
                        <ExpandableMarkdown className="prose-sm" content={subagentPrompt} maxLength={500} />
                      </div>
                    </div>
                  )}

                  {/* 2. ACTIVITY — Agent child tool calls */}
                  {childToolCalls && childToolCalls.length > 0 && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-foreground/50 uppercase tracking-wide">
                        Activity
                      </h4>
                      <div className="space-y-0.5 pl-3 border-l-2 border-border">
                        {childToolCalls.map((child, idx) => (
                          <ChildToolCall
                            key={`child-${child.toolUseBlock?.id || idx}`}
                            toolUseBlock={child.toolUseBlock}
                            resultBlock={child.resultBlock}
                          />
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Loading indicator */}
                  {isLoading && (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      <span>
                        {childToolCalls && childToolCalls.length > 0
                          ? "Processing..."
                          : "Waiting for result…"}
                      </span>
                    </div>
                  )}

                  {/* 3. RESULT */}
                  {hasActualResult && (
                    <div>
                      <h4 className="text-sm font-medium text-foreground/50 uppercase tracking-wide">
                        Result {isError && <span className="text-destructive">(Error)</span>}
                      </h4>
                      <div className={cn(
                        "rounded-lg p-3 mt-1 overflow-x-auto border text-base",
                        isError
                          ? "bg-destructive/5 dark:bg-destructive/8 border-destructive/20 dark:border-destructive/25"
                          : "bg-muted/20 border-border/50"
                      )}>
                        <ExpandableMarkdown
                          className="prose-sm"
                          content={extractTextFromResultContent(toolResultBlock?.content as unknown)}
                          maxLength={1000}
                        />
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                // Default tool rendering
                isExpanded && (
                  <div className="px-3 pb-3 space-y-3 bg-muted/20 rounded-b-lg">
                    {toolUseBlock?.input && (
                      <div>
                        <h4 className="text-sm font-medium text-foreground/60 mb-1">Input</h4>
                        <div className="bg-code-bg rounded-lg p-3 overflow-x-auto border border-border/50">
                          <pre className="text-code-foreground text-sm">
                            {formatToolInput(JSON.stringify(toolUseBlock.input))}
                          </pre>
                        </div>
                      </div>
                    )}

                    {isToolResult && (
                      <div>
                        <h4 className="text-sm font-medium text-foreground/60 mb-1">
                          Result {isError && <span className="text-destructive">(Error)</span>}
                        </h4>
                        <div
                          className={cn(
                            "rounded-lg p-3 overflow-x-auto text-foreground",
                            isError && "bg-destructive/5 dark:bg-destructive/8 border border-destructive/20 dark:border-destructive/25"
                          )}
                        >
                          <ExpandableMarkdown
                            className="prose-sm"
                            content={
                              typeof toolResultBlock?.content === "string"
                                ? (toolResultBlock?.content as string)
                                : JSON.stringify(toolResultBlock?.content ?? "")
                            }
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }
);

ToolMessage.displayName = "ToolMessage";
