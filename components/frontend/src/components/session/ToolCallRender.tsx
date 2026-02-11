"use client";

import { useState } from "react";
import { useDefaultTool, useCopilotContext } from "@copilotkit/react-core";
import { useAgent } from "@copilotkit/react-core/v2";
import { Loader2, Check, X, ChevronRight, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatToolName, toolSummary } from "./tool-call-utils";
import { InlineToolRow } from "./InlineToolRow";

// ─── Tool nesting (data-driven via message ordering) ──────────────────
//
// AG-UI messages embed tool calls inside assistant messages as
//   { id, type: "function", function: { name, arguments } }
// and results as separate tool-role messages with `toolCallId`.
//
// We flatten all tool calls into a positional list and infer nesting
// from message ordering: a tool call that sits between another TC and
// its result is treated as a child of that parent TC.
//
// The render function (`useDefaultTool`) is called once per tool call.
// Inside the component we use `useAgent().agent.messages` to determine:
//   1. Is this tool call a *child* of another tool call? → return null
//   2. Does this tool call *have* children? → render them inline

function ToolCallRender({
  name,
  args,
  status,
  result,
}: {
  name: string;
  args: Record<string, unknown>;
  status: string;
  result: unknown;
}) {
  const [expanded, setExpanded] = useState(true);
  const { agentSession } = useCopilotContext();
  const agentId = agentSession?.agentName ?? "session";
  const { agent: { messages } } = useAgent({ agentId });

  const isComplete = status === "complete";
  const isError = isComplete && result instanceof Error;
  const isSuccess = isComplete && !isError;
  const isLoading = !isComplete;

  // ── Flatten tool calls from AG-UI assistant messages ──────────────
  type FlatTC = { id: string; name: string; args: string; msgIndex: number };

  const toolCallEntries: FlatTC[] = messages.flatMap((m, idx) =>
    m.role === "assistant" && "toolCalls" in m && Array.isArray(m.toolCalls)
      ? (m.toolCalls as { id: string; type: string; function: { name: string; arguments: string } }[]).map((tc) => ({
          id: tc.id,
          name: tc.function.name,
          args: tc.function.arguments,
          msgIndex: idx,
        }))
      : [],
  );

  // Map tool-call ID → tool result message (role: "tool")
  const toolResults = new Map<string, { content: string; msgIndex: number }>();
  messages.forEach((m, idx) => {
    if (m.role === "tool" && "toolCallId" in m) {
      toolResults.set(
        m.toolCallId as string,
        { content: (m as { content: string }).content, msgIndex: idx },
      );
    }
  });

  // Find our tool call by matching name + normalised args.
  // AG-UI messages keep the raw `function.arguments` string (often pretty-
  // printed with spaces after colons), while `args` here is already parsed.
  // We normalise both sides so the comparison isn't thrown off by whitespace.
  const normalise = (raw: string): string => {
    if (!raw || raw.trim() === "") return "{}";
    try {
      return JSON.stringify(JSON.parse(raw));
    } catch {
      return raw;
    }
  };

  const argsStr = JSON.stringify(args);
  const thisTC = toolCallEntries.find(
    (tc) => tc.name === name && normalise(tc.args) === argsStr,
  );

  // ── Child detection: if we sit between another TC and its result,
  //    we're a nested child — the parent's render will display us.
  if (thisTC) {
    const isChild = toolCallEntries.some((tc) => {
      if (tc.id === thisTC.id) return false;
      const tcResult = toolResults.get(tc.id);
      return (
        tcResult &&
        tc.msgIndex < thisTC.msgIndex &&
        tcResult.msgIndex > thisTC.msgIndex
      );
    });
    // Return a ref-based element that hides the CopilotKit wrapper div.
    // Returning `null` would still leave the wrapper (with its margins) in the DOM.
    if (isChild)
      return (
        <div
          ref={(el) => {
            if (el?.parentElement) el.parentElement.style.display = "none";
          }}
        />
      );
  }

  // ── Find children: TCs between our start and our result ───────────
  const thisResult = thisTC ? toolResults.get(thisTC.id) : undefined;
  const childTCs =
    thisTC && thisResult
      ? toolCallEntries.filter(
          (tc) =>
            tc.id !== thisTC.id &&
            tc.msgIndex > thisTC.msgIndex &&
            tc.msgIndex < thisResult.msgIndex,
        )
      : [];

  if (childTCs.length > 0) {
    const displayName = formatToolName(name);
    const summary = toolSummary(name, args);
    const toolCount = childTCs.filter((c) => c.name !== "Task").length;

    const childList = childTCs.map((child) => {
      const childResult = toolResults.get(child.id);
      return {
        id: child.id,
        name: child.name,
        args: (child.args ? JSON.parse(child.args) : {}) as Record<string, unknown>,
        status: childResult ? "complete" : "inProgress",
        result: childResult?.content,
      };
    });

    return (
      <div className="mb-0">
        <div
          className="flex items-center gap-1.5 cursor-pointer hover:bg-muted/50 rounded py-0.5 px-0"
          onClick={() => setExpanded(!expanded)}
        >
          {isLoading && <Loader2 className="w-3 h-3 text-blue-500 animate-spin flex-shrink-0" />}
          {isSuccess && <Check className="w-3 h-3 text-green-500 flex-shrink-0" />}
          {isError && <X className="w-3 h-3 text-red-500 flex-shrink-0" />}

          <span
            className={cn(
              "text-xs font-medium px-1.5 py-0.5 rounded text-white flex-shrink-0",
              isLoading && "bg-blue-500",
              isError && "bg-red-600",
              isSuccess && "bg-green-600",
            )}
          >
            {displayName}
          </span>

          <span className="text-[11px] text-muted-foreground truncate flex-1">{summary}</span>

          {expanded ? (
            <ChevronDown className="w-3 h-3 text-muted-foreground flex-shrink-0" />
          ) : (
            <ChevronRight className="w-3 h-3 text-muted-foreground flex-shrink-0" />
          )}

          {toolCount > 0 && (
            <span className="text-[10px] text-muted-foreground/60 flex-shrink-0">
              {toolCount} tool{toolCount !== 1 ? "s" : ""}
            </span>
          )}
        </div>

        {expanded && (
          <div className="mt-0.5 border-l-2 border-muted-foreground/25 pl-1 space-y-0">
            {childList.map((child) =>
              child.name === "Task" ? (
                <div
                  key={child.id}
                  className="flex items-center gap-1.5 ml-4 py-0.5 text-[10px] text-muted-foreground/50"
                >
                  {child.status !== "complete" ? (
                    <Loader2 className="w-2.5 h-2.5 animate-spin flex-shrink-0" />
                  ) : (
                    <Check className="w-2.5 h-2.5 flex-shrink-0" />
                  )}
                  <span className="border-b border-muted-foreground/15 flex-1" />
                  <span className="italic px-1">sub-agent</span>
                  <span className="border-b border-muted-foreground/15 flex-1" />
                </div>
              ) : (
                <InlineToolRow
                  key={child.id}
                  name={child.name}
                  args={child.args}
                  status={child.status}
                  result={child.result}
                  indent
                />
              ),
            )}
          </div>
        )}
      </div>
    );
  }

  // ── Standalone tool (no children) ─────────────────────────────────
  return (
    <div className="mb-0">
      <InlineToolRow name={name} args={args} status={status} result={result} />
    </div>
  );
}

/**
 * Registers the default tool renderer once inside the CopilotKit context.
 * Separate component so it isn't duplicated across mobile/desktop views.
 */
export function DefaultToolRegistration() {
  useDefaultTool({
    render: ({ name, args, status, result }) => (
      <ToolCallRender
        name={name}
        args={(args ?? {}) as Record<string, unknown>}
        status={status}
        result={result}
      />
    ),
  });
  return null;
}
