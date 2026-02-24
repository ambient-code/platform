"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import type { ThinkingBlock } from "@/types/agentic-session";

export type ThinkingMessageProps = {
  block: ThinkingBlock;
  streaming?: boolean;
  className?: string;
};

export const ThinkingMessage: React.FC<ThinkingMessageProps> = ({ block, streaming, className }) => {
  const [expanded, setExpanded] = useState(false);
  const text = block.thinking || "";

  return (
    <div className={cn("py-1", className)}>
      <button
        type="button"
        className="flex items-center gap-1.5 w-full text-left group"
        onClick={() => setExpanded((e) => !e)}
      >
        <ChevronRight
          className={cn(
            "h-3.5 w-3.5 shrink-0 text-muted-foreground/60 transition-transform duration-150",
            expanded && "rotate-90",
          )}
        />
        <span className="text-xs font-medium text-muted-foreground/70">
          Thinking
          {streaming && !expanded && (
            <span className="ml-1 animate-pulse">...</span>
          )}
        </span>
        {!expanded && text && (
          <span className="text-xs text-muted-foreground/50 truncate min-w-0">
            &mdash; {text}
          </span>
        )}
      </button>

      {expanded && (
        <div className="mt-1 ml-5">
          <pre className="text-xs text-muted-foreground/60 whitespace-pre-wrap break-words leading-relaxed">
            {text}
            {streaming && <span className="animate-pulse">&#9608;</span>}
          </pre>
        </div>
      )}
    </div>
  );
};

export default ThinkingMessage;
