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
  // Remove mcp__ prefix and format nicely
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

  // Match Message.tsx rendering so headers/code look correct
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
        <code className="bg-muted px-1 py-0.5 rounded text-xs" {...(props as React.HTMLAttributes<HTMLElement>)}>
          {children}
        </code>
      ) : (
        <pre className="bg-slate-950 dark:bg-black text-slate-50 p-2 rounded text-xs overflow-x-auto">
          <code className={className} {...(props as React.HTMLAttributes<HTMLElement>)}>
            {children}
          </code>
        </pre>
      );
    },
    p: ({ children }) => <p className="text-muted-foreground leading-relaxed mb-2 text-sm">{children}</p>,
    h1: ({ children }) => <h1 className="text-lg font-bold text-foreground mb-2">{children}</h1>,
    h2: ({ children }) => <h2 className="text-md font-semibold text-foreground mb-2">{children}</h2>,
    h3: ({ children }) => <h3 className="text-sm font-medium text-foreground mb-1">{children}</h3>,
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
            className="text-xs px-2 py-1 rounded border bg-card hover:bg-muted/50 text-foreground/80"
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
    hash |= 0; // Convert to 32bit integer
  }
  return Math.abs(hash);
};

const getColorClassesForName = (name: string) => {
  const colorChoices = [
    { avatarBg: "bg-purple-600", badgeBg: "bg-purple-600", cardBg: "bg-purple-50 dark:bg-purple-950/50", border: "border-purple-200 dark:border-purple-800", badgeText: "text-purple-700 dark:text-purple-300", badgeBorder: "border-purple-200 dark:border-purple-800" },
    { avatarBg: "bg-blue-600", badgeBg: "bg-blue-600", cardBg: "bg-blue-50 dark:bg-blue-950/50", border: "border-blue-200 dark:border-blue-800", badgeText: "text-blue-700 dark:text-blue-300", badgeBorder: "border-blue-200 dark:border-blue-800" },
    { avatarBg: "bg-emerald-600", badgeBg: "bg-emerald-600", cardBg: "bg-emerald-50 dark:bg-emerald-950/50", border: "border-emerald-200 dark:border-emerald-800", badgeText: "text-emerald-700 dark:text-emerald-300", badgeBorder: "border-emerald-200 dark:border-emerald-800" },
    { avatarBg: "bg-teal-600", badgeBg: "bg-teal-600", cardBg: "bg-teal-50 dark:bg-teal-950/50", border: "border-teal-200 dark:border-teal-800", badgeText: "text-teal-700 dark:text-teal-300", badgeBorder: "border-teal-200 dark:border-teal-800" },
    { avatarBg: "bg-cyan-600", badgeBg: "bg-cyan-600", cardBg: "bg-cyan-50 dark:bg-cyan-950/50", border: "border-cyan-200 dark:border-cyan-800", badgeText: "text-cyan-700 dark:text-cyan-300", badgeBorder: "border-cyan-200 dark:border-cyan-800" },
    { avatarBg: "bg-sky-600", badgeBg: "bg-sky-600", cardBg: "bg-sky-50 dark:bg-sky-950/50", border: "border-sky-200 dark:border-sky-800", badgeText: "text-sky-700 dark:text-sky-300", badgeBorder: "border-sky-200 dark:border-sky-800" },
    { avatarBg: "bg-indigo-600", badgeBg: "bg-indigo-600", cardBg: "bg-indigo-50 dark:bg-indigo-950/50", border: "border-indigo-200 dark:border-indigo-800", badgeText: "text-indigo-700 dark:text-indigo-300", badgeBorder: "border-indigo-200 dark:border-indigo-800" },
    { avatarBg: "bg-fuchsia-600", badgeBg: "bg-fuchsia-600", cardBg: "bg-fuchsia-50 dark:bg-fuchsia-950/50", border: "border-fuchsia-200 dark:border-fuchsia-800", badgeText: "text-fuchsia-700 dark:text-fuchsia-300", badgeBorder: "border-fuchsia-200 dark:border-fuchsia-800" },
    { avatarBg: "bg-rose-600", badgeBg: "bg-rose-600", cardBg: "bg-rose-50 dark:bg-rose-950/50", border: "border-rose-200 dark:border-rose-800", badgeText: "text-rose-700 dark:text-rose-300", badgeBorder: "border-rose-200 dark:border-rose-800" },
    { avatarBg: "bg-amber-600", badgeBg: "bg-amber-600", cardBg: "bg-amber-50 dark:bg-amber-950/50", border: "border-amber-200 dark:border-amber-800", badgeText: "text-amber-700 dark:text-amber-300", badgeBorder: "border-amber-200 dark:border-amber-800" },
  ];
  const idx = hashStringToNumber(name) % colorChoices.length;
  return colorChoices[idx];
};

// Helper to convert Python literal to JSON-parseable string
const pythonLiteralToJson = (pythonStr: string): string => {
  // This handles Python dict/list notation like [{'type': 'text', 'text': '...'}]
  // Use a state machine to properly handle quotes and escape sequences
  
  let result = '';
  let inString = false;
  let stringChar = '';
  let escaped = false;
  
  for (let i = 0; i < pythonStr.length; i++) {
    const char = pythonStr[i];
    
    if (escaped) {
      // Handle escape sequences
      if (inString) {
        // Inside string - convert Python escapes to JSON escapes
        if (char === "'") {
          // Python: \' → JSON: ' (no escape needed in double-quoted JSON string)
          result += "'";
        } else if (char === '"') {
          // Python: \" → JSON: \" (keep escape)
          result += '\\"';
        } else {
          // All other escapes (\n, \t, \\, etc.) are valid in both
          result += '\\' + char;
        }
      } else {
        // Outside string - just copy
        result += '\\' + char;
      }
      escaped = false;
      continue;
    }
    
    if (char === '\\') {
      escaped = true;
      continue;
    }
    
    // Handle quotes
    if (char === "'" || char === '"') {
      if (!inString) {
        // Starting a string
        inString = true;
        stringChar = char;
        result += '"'; // Always use double quotes in JSON
      } else if (char === stringChar) {
        // Ending a string
        inString = false;
        stringChar = '';
        result += '"';
      } else {
        // Different quote inside string
        if (char === '"') {
          // Double quote inside single-quoted Python string
          result += '\\"'; // Must escape in JSON
        } else {
          // Single quote inside double-quoted string
          result += "'"; // No escape needed
        }
      }
      continue;
    }
    
    // If we're in a string, just copy characters
    if (inString) {
      result += char;
      continue;
    }
    
    // Handle Python keywords outside strings
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
    // If string, try to parse as JSON/Python first (handles stringified arrays/objects)
    if (typeof content === "string") {
      // Try parsing if it looks like JSON/Python
      if (content.trim().startsWith("[") || content.trim().startsWith("{")) {
        try {
          // FIRST: Try parsing as valid JSON (backend now sends proper JSON)
          const parsed = JSON.parse(content);
          return extractTextFromResultContent(parsed);
        } catch {
          // FALLBACK: Try converting Python notation to JSON (for old sessions)
          try {
            const jsonStr = pythonLiteralToJson(content);
            const parsed = JSON.parse(jsonStr);
            return extractTextFromResultContent(parsed);
          } catch {
            // LAST RESORT: Can't parse, just return the raw string
            // This handles cases where the content is malformed or uses unknown syntax
            console.warn('Failed to parse result content, showing raw text');
            return content;
          }
        }
      }
      return content;
    }
    
    // Handle arrays of text blocks
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
    
    // Handle nested content arrays
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
  
  // WebSearch - show query
  if (toolName.toLowerCase().includes("websearch") || toolName.toLowerCase().includes("web_search")) {
    const query = input.query as string | undefined;
    if (query) return `Searching the web for "${query}"`;
  }
  
  // FileRead - show file path
  if (toolName.toLowerCase().includes("read") && (input.file || input.path || input.target_file)) {
    const file = (input.file || input.path || input.target_file) as string;
    return `Reading ${file}`;
  }
  
  // FileWrite - show file path
  if (toolName.toLowerCase().includes("write") && (input.file || input.path || input.target_file)) {
    const file = (input.file || input.path || input.target_file) as string;
    return `Writing to ${file}`;
  }
  
  // Grep - show pattern and path
  if (toolName.toLowerCase().includes("grep") || toolName.toLowerCase().includes("search")) {
    const pattern = input.pattern as string | undefined;
    const path = input.path as string | undefined;
    if (pattern && path) return `Searching for "${pattern}" in ${path}`;
    if (pattern) return `Searching for "${pattern}"`;
  }
  
  // Command execution
  if (toolName.toLowerCase().includes("command") || toolName.toLowerCase().includes("terminal")) {
    const command = input.command as string | undefined;
    if (command) {
      const truncated = command.length > 50 ? command.substring(0, 50) + "..." : command;
      return `Running: ${truncated}`;
    }
  }
  
  // Fallback: show first string value from input (often contains the main parameter)
  const firstStringValue = Object.values(input).find(v => typeof v === 'string' && v.length > 0) as string | undefined;
  if (firstStringValue) {
    const truncated = firstStringValue.length > 60 ? firstStringValue.substring(0, 60) + "..." : firstStringValue;
    return truncated;
  }
  
  // Last resort: show formatted tool name
  return formatToolName(toolName);
};

// Child Tool Call component for hierarchical rendering (collapsed by default)
type ChildToolCallProps = {
  toolUseBlock?: ToolUseBlock;
  resultBlock?: ToolResultBlock;
};

const ChildToolCall: React.FC<ChildToolCallProps> = ({ toolUseBlock, resultBlock }) => {
  const [expanded, setExpanded] = useState(false);
  
  // Check if result has actual content (same logic as parent tool)
  const hasActualResult = Boolean(
    resultBlock && 
    resultBlock.content !== undefined && 
    resultBlock.content !== null &&
    (() => {
      const content = resultBlock.content;
      // Empty string
      if (content === "") return false;
      // Empty array
      if (Array.isArray(content) && content.length === 0) return false;
      // Empty object
      if (typeof content === 'object' && !Array.isArray(content) && Object.keys(content).length === 0) return false;
      // String that only contains whitespace or quotes
      if (typeof content === 'string' && content.trim() === '') return false;
      if (typeof content === 'string' && (content === '""' || content === "''")) return false;
      // Has actual content
      return true;
    })()
  );
  
  const isError = resultBlock?.is_error === true;
  const isSuccess = hasActualResult && !isError;
  const isPending = !hasActualResult && !isError;
  
  const toolName = toolUseBlock?.name || "unknown_tool";
  
  // Parse input - it might be a string that needs JSON parsing or already an object
  let toolInput: Record<string, unknown> | undefined;
  if (toolUseBlock?.input) {
    if (typeof toolUseBlock.input === 'string') {
      try {
        toolInput = JSON.parse(toolUseBlock.input) as Record<string, unknown>;
      } catch {
        // If parsing fails, treat the string as a single value
        toolInput = { value: toolUseBlock.input };
      }
    } else {
      toolInput = toolUseBlock.input as Record<string, unknown>;
    }
  }
  
  // Generate smart collapsed summary - ALWAYS show the query/input, not the result
  // Result should only be visible when expanded
  const collapsedSummary = generateToolSummary(toolName, toolInput);
  
  return (
    <div className="py-1">
      <div 
        className="flex items-center gap-2 cursor-pointer hover:bg-muted/30 rounded px-2 py-1"
        onClick={() => setExpanded(!expanded)}
      >
        {isError && <X className="w-3 h-3 text-red-500 flex-shrink-0" />}
        {isSuccess && <Check className="w-3 h-3 text-green-500 flex-shrink-0" />}
        {isPending && <Loader2 className="w-3 h-3 animate-spin text-blue-500 flex-shrink-0" />}
        
        <Badge variant="secondary" className="text-[10px] px-1.5 py-0.5 flex-shrink-0">
          {formatToolName(toolName)}
        </Badge>
        
        {/* Collapsed summary */}
        {!expanded && (
          <span className="text-[11px] text-muted-foreground truncate flex-1">
            {collapsedSummary}
          </span>
        )}
        
        <ChevronRight className={cn(
          "w-3 h-3 text-muted-foreground transition-transform flex-shrink-0",
          expanded && "rotate-90"
        )} />
      </div>
      
      {expanded && (
        <div className="mt-1 ml-5 text-xs space-y-2 bg-muted/20 rounded p-2 border border-border">
          {toolInput && Object.keys(toolInput).length > 0 && (
            <div>
              <div className="font-medium text-foreground/70 mb-1">Input</div>
              <pre className="text-[10px] overflow-x-auto text-muted-foreground">
                {JSON.stringify(toolInput, null, 2)}
              </pre>
            </div>
          )}
          {resultBlock?.content && (
            <div>
              <div className="font-medium text-foreground/70 mb-1">
                Result {isError && <span className="text-red-600">(Error)</span>}
              </div>
              {/* Render result as markdown for better formatting */}
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
    
    // Check if result has actual content (not just empty object/array/string)
    const hasActualResult = Boolean(
      toolResultBlock && 
      toolResultBlock.content !== undefined && 
      toolResultBlock.content !== null &&
      (() => {
        const content = toolResultBlock.content;
        // Empty string
        if (content === "") return false;
        // Empty array
        if (Array.isArray(content) && content.length === 0) return false;
        // Empty object
        if (typeof content === 'object' && !Array.isArray(content) && Object.keys(content).length === 0) return false;
        // String that only contains whitespace or quotes
        if (typeof content === 'string' && content.trim() === '') return false;
        if (typeof content === 'string' && (content === '""' || content === "''")) return false;
        // Has actual content
        return true;
      })()
    );
    
    const isToolCall = Boolean(toolUseBlock && !hasActualResult);
    const isToolResult = hasActualResult;

    // For tool calls/results, show collapsible interface
    const toolName = formatToolName(toolUseBlock?.name);
    const isError = toolResultBlock?.is_error === true;
    const isLoading = isToolCall && !isError; // Tool call without result is loading, unless there's an error
    const isSuccess = isToolResult && !isError;

    // Subagent detection and data
    const inputData = (toolUseBlock?.input ?? undefined) as unknown as Record<string, unknown> | undefined;
    const subagentType = (inputData?.subagent_type as string) || undefined;
    const subagentDescription = (inputData?.description as string) || undefined;
    const subagentPrompt = (inputData?.prompt as string) || undefined;
    const isSubagent = Boolean(subagentType);
    const subagentClasses = subagentType ? getColorClassesForName(subagentType) : undefined;
    const displayName = isSubagent ? subagentType : toolName;

    // Compact mode for simple tool calls (non-subagent)
    const isCompact = !isSubagent;

    const formattedTime = formatTimestamp(timestamp);

    return (
      <div ref={ref} className={cn(isCompact ? "mb-1" : "mb-4", className)} {...props}>
        <div className="flex items-start space-x-3">
          {/* Avatar */}
          <div className="flex-shrink-0">
            {isSubagent ? (
              <div className={cn("w-8 h-8 rounded-full flex items-center justify-center", subagentClasses?.avatarBg)}>
                <span className="text-white text-xs font-semibold">
                  {getInitials(subagentType)}
                </span>
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full flex items-center justify-center bg-purple-600">
                <Cog className="w-4 h-4 text-white" />
              </div>
            )}
          </div>

          {/* Tool Message Content */}
          <div className="flex-1 min-w-0">
            {/* Timestamp */}
            {formattedTime && (
              <div className="text-[10px] text-muted-foreground/60 mb-1">
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
                  "flex items-center justify-between cursor-pointer hover:bg-muted/50 transition-colors",
                  isCompact ? "py-1 px-0" : "p-3"
                )}
                onClick={() => setIsExpanded(!isExpanded)}
              >
                <div className={cn("flex items-center flex-1 min-w-0", isCompact ? "space-x-1.5" : "space-x-2")}>
                  {/* Status Icon */}
                  {!isCompact && (
                    <div className="flex-shrink-0">
                      {isLoading && (
                        <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                      )}
                      {isSuccess && <Check className="w-4 h-4 text-green-500" />}
                      {isError && <X className="w-4 h-4 text-red-500" />}
                    </div>
                  )}
                  {isCompact && (
                    <div className="flex-shrink-0">
                      {isLoading && (
                        <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />
                      )}
                      {isSuccess && <Check className="w-3 h-3 text-green-500" />}
                      {isError && <X className="w-3 h-3 text-red-500" />}
                    </div>
                  )}

                  {/* Tool Name Badge */}
                  <div className="flex-shrink-0">
                    <Badge
                      className={cn(
                        "text-xs text-white",
                        isLoading && "bg-blue-500",
                        isError && "bg-red-600",
                        isSuccess && "bg-green-600",
                        isSubagent && subagentClasses?.badgeBg,
                        isCompact && "!py-0 px-1.5 leading-tight"
                      )}
                    >
                      {displayName}
                    </Badge>
                  </div>

                  {/* Title/Description - Always visible (collapsed or expanded) */}
                  <div className="flex-1 min-w-0 text-sm text-muted-foreground truncate">
                    {isSubagent ? (
                      // Agent: Show description (title)
                      <span className="truncate">
                        {subagentDescription || subagentPrompt || "Working..."}
                      </span>
                    ) : (
                      // Regular tool: Show query/input summary
                      <span className="truncate">
                        {generateToolSummary(toolUseBlock?.name || "", inputData)}
                      </span>
                    )}
                  </div>

                  {/* Expand/Collapse Icon */}
                  <div className="flex-shrink-0">
                    {isExpanded ? (
                      <ChevronDown className={cn(isCompact ? "w-3 h-3" : "w-4 h-4", "text-muted-foreground/60")} />
                    ) : (
                      <ChevronRight className={cn(isCompact ? "w-3 h-3" : "w-4 h-4", "text-muted-foreground/60")} />
                    )}
                  </div>
                </div>
              </div>

              {/* Subagent primary content - REORDERED: Input → Activity → Result */}
              {isSubagent && isExpanded ? (
                <div className="px-3 pb-3 space-y-3">
                  {/* 1. INPUT - Show when expanded */}
                  {subagentPrompt && (
                    <div className="space-y-2">
                      <h4 className="text-xs font-medium text-foreground/60 uppercase tracking-wide">
                        Prompt
                      </h4>
                      <div className="rounded p-2 overflow-x-auto bg-muted/20 border border-border text-xs text-muted-foreground">
                        <ExpandableMarkdown className="prose-sm" content={subagentPrompt} maxLength={500} />
                      </div>
                    </div>
                  )}
                  
                  {/* 2. ACTIVITY - Agent child tool calls */}
                  {childToolCalls && childToolCalls.length > 0 && (
                    <div className="space-y-2">
                      <h4 className="text-xs font-medium text-foreground/60 uppercase tracking-wide">
                        Activity
                      </h4>
                      <div className="space-y-1 pl-2 border-l-2 border-purple-200 dark:border-purple-800">
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
                  
                  {/* Loading indicator when waiting for result */}
                  {isLoading && (
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Loader2 className="w-3 h-3 animate-spin" />
                      <span>
                        {childToolCalls && childToolCalls.length > 0 
                          ? "Processing..." 
                          : "Waiting for result…"}
                      </span>
                    </div>
                  )}

                  {/* 3. RESULT - Only show if there's actual content */}
                  {hasActualResult && (
                    <div>
                      <h4 className="text-xs font-medium text-foreground/60 uppercase tracking-wide">
                        Result {isError && <span className="text-red-600">(Error)</span>}
                      </h4>
                      <div className={cn(
                        "rounded p-2 mt-1 overflow-x-auto border text-xs",
                        isError 
                          ? "bg-red-50 dark:bg-red-950/50 border-red-200 dark:border-red-800"
                          : "bg-muted/30 border-border"
                      )}>
                        {/* CRITICAL: Render result as markdown for better formatting */}
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
                // Default tool rendering (existing behavior)
                isExpanded && (
                  <div className="px-3 pb-3 space-y-3 bg-muted/50">
                    {toolUseBlock?.input && (
                      <div>
                        <h4 className="text-xs font-medium text-foreground/80 mb-1">Input</h4>
                        <div className="bg-slate-950 dark:bg-black rounded text-xs p-2 overflow-x-auto">
                          <pre className="text-gray-100">
                            {formatToolInput(JSON.stringify(toolUseBlock.input))}
                          </pre>
                        </div>
                      </div>
                    )}

                    {isToolResult && (
                      <div>
                        <h4 className="text-xs font-medium text-foreground/80 mb-1">
                          Result {isError && <span className="text-red-600 dark:text-red-400">(Error)</span>}
                        </h4>
                        <div
                          className={cn(
                            "rounded p-2 overflow-x-auto text-foreground",
                            isError && "bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800"
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