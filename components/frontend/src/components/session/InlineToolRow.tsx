"use client";

import { useState } from "react";
import { Loader2, Check, X, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatToolName, toolSummary } from "./tool-call-utils";

type InlineToolRowProps = {
  name: string;
  args: Record<string, unknown>;
  status: string;
  result: unknown;
  indent?: boolean;
};

export function InlineToolRow({ name, args, status, result, indent }: InlineToolRowProps) {
  const [expanded, setExpanded] = useState(false);
  const isComplete = status === "complete";
  const isError = isComplete && result instanceof Error;
  const isSuccess = isComplete && !isError;
  const isLoading = !isComplete;
  const displayName = formatToolName(name);
  const summary = toolSummary(name, args);

  const resultText =
    isComplete && result != null
      ? typeof result === "string"
        ? result
        : JSON.stringify(result, null, 2)
      : null;

  return (
    <div className={cn(indent && "ml-4")}>
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

        <ChevronRight
          className={cn(
            "w-3 h-3 text-muted-foreground transition-transform flex-shrink-0",
            expanded && "rotate-90",
          )}
        />
      </div>

      {expanded && (
        <div className="mt-1 ml-5 text-xs space-y-2 bg-muted/20 rounded p-2 border border-border">
          {Object.keys(args).length > 0 && (
            <div>
              <div className="font-medium text-foreground/70 mb-1">Input</div>
              <pre className="text-[10px] overflow-x-auto text-muted-foreground">
                {JSON.stringify(args, null, 2)}
              </pre>
            </div>
          )}
          {resultText && (
            <div>
              <div className="font-medium text-foreground/70 mb-1">
                Result {isError && <span className="text-red-600">(Error)</span>}
              </div>
              <pre className="text-[10px] overflow-x-auto text-muted-foreground whitespace-pre-wrap break-words">
                {resultText.length > 2000 ? resultText.substring(0, 2000) + "\n… (truncated)" : resultText}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
