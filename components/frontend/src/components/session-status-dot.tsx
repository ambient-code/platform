"use client";

import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { AgenticSessionPhase } from "@/types/agentic-session";

type SessionStatusDotProps = {
  phase: AgenticSessionPhase | string;
  className?: string;
};

const DOT_COLORS: Record<string, string> = {
  Running: "bg-blue-500",
  Completed: "bg-gray-400",
  Stopped: "bg-gray-400",
  Failed: "bg-red-500",
  Pending: "bg-orange-400",
  Creating: "bg-orange-400",
  Stopping: "bg-orange-400",
};

const DOT_ANIMATIONS: Record<string, string> = {
  Running: "animate-pulse",
  Creating: "animate-pulse",
  Stopping: "animate-pulse",
};

export function SessionStatusDot({ phase, className }: SessionStatusDotProps) {
  const color = DOT_COLORS[phase] || "bg-gray-400";
  const animation = DOT_ANIMATIONS[phase] || "";

  return (
    <TooltipProvider delayDuration={300}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            className={cn(
              "inline-block w-2.5 h-2.5 rounded-full flex-shrink-0",
              color,
              animation,
              className
            )}
            aria-label={`Session: ${phase}`}
          />
        </TooltipTrigger>
        <TooltipContent side="top">
          <p className="text-xs">Session: {phase}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
