"use client";

import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import {
  Loader2,
  CheckCircle2,
  XCircle,
  Circle,
  HelpCircle,
} from "lucide-react";
import type { AgentStatus } from "@/types/agentic-session";

type AgentStatusIndicatorProps = {
  status: AgentStatus;
  compact?: boolean;
  className?: string;
};

export function AgentStatusIndicator({
  status,
  compact = false,
  className,
}: AgentStatusIndicatorProps) {
  switch (status) {
    case "working":
      return (
        <div className={cn("flex items-center gap-1.5", className)}>
          <Loader2
            className={cn(
              "animate-spin text-blue-500",
              compact ? "w-3.5 h-3.5" : "w-4 h-4"
            )}
          />
          {!compact && (
            <span className="text-sm text-blue-600 dark:text-blue-400 font-medium">
              Working
            </span>
          )}
        </div>
      );

    case "waiting_input":
      return (
        <Badge
          className={cn(
            "bg-amber-500 hover:bg-amber-500 text-white animate-pulse gap-1",
            compact && "px-1.5 py-0.5 text-[10px]",
            className
          )}
        >
          <HelpCircle className={cn(compact ? "w-3 h-3" : "w-3.5 h-3.5")} />
          {compact ? "Input" : "Needs Input"}
        </Badge>
      );

    case "completed":
      return (
        <div className={cn("flex items-center gap-1.5", className)}>
          <CheckCircle2
            className={cn(
              "text-green-500",
              compact ? "w-3.5 h-3.5" : "w-4 h-4"
            )}
          />
          {!compact && (
            <span className="text-sm text-green-600 dark:text-green-400">
              Completed
            </span>
          )}
        </div>
      );

    case "failed":
      return (
        <div className={cn("flex items-center gap-1.5", className)}>
          <XCircle
            className={cn(
              "text-red-500",
              compact ? "w-3.5 h-3.5" : "w-4 h-4"
            )}
          />
          {!compact && (
            <span className="text-sm text-red-600 dark:text-red-400">
              Failed
            </span>
          )}
        </div>
      );

    case "idle":
      return (
        <div className={cn("flex items-center gap-1.5", className)}>
          <Circle
            className={cn(
              "text-gray-400",
              compact ? "w-3.5 h-3.5" : "w-4 h-4"
            )}
          />
          {!compact && (
            <span className="text-sm text-muted-foreground">Idle</span>
          )}
        </div>
      );
  }
}
