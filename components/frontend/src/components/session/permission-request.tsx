"use client";

import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ShieldCheck, ShieldX, ShieldAlert } from "lucide-react";
import { formatTimestamp } from "@/lib/format-timestamp";
import {
  hasToolResult,
  type ToolUseBlock,
  type ToolResultBlock,
  type PermissionRequestInput,
} from "@/types/agentic-session";

export type PermissionRequestMessageProps = {
  toolUseBlock: ToolUseBlock;
  resultBlock?: ToolResultBlock;
  timestamp?: string;
  onSubmitAnswer?: (formattedAnswer: string) => Promise<void>;
  isNewest?: boolean;
};

function isPermissionRequestInput(
  input: Record<string, unknown>
): input is PermissionRequestInput {
  return "tool_name" in input && "key" in input;
}

type PermissionStatus = "pending" | "approved" | "denied";

function deriveStatus(resultBlock?: ToolResultBlock): PermissionStatus {
  if (!hasToolResult(resultBlock)) return "pending";
  const content = resultBlock?.content;
  if (typeof content !== "string") return "denied";
  try {
    return JSON.parse(content).approved === true ? "approved" : "denied";
  } catch {
    return "denied";
  }
}

const STATUS_CONFIG: Record<PermissionStatus, {
  icon: typeof ShieldCheck;
  avatarClass: string;
  borderClass: string;
}> = {
  pending: {
    icon: ShieldAlert,
    avatarClass: "bg-amber-500",
    borderClass: "border-l-amber-500 bg-amber-50/30 dark:bg-amber-950/10",
  },
  approved: {
    icon: ShieldCheck,
    avatarClass: "bg-green-600",
    borderClass: "border-l-green-500 bg-green-50/30 dark:bg-green-950/10",
  },
  denied: {
    icon: ShieldX,
    avatarClass: "bg-red-600",
    borderClass: "border-l-red-500 bg-red-50/30 dark:bg-red-950/10",
  },
};

export const PermissionRequestMessage: React.FC<
  PermissionRequestMessageProps
> = ({ toolUseBlock, resultBlock, timestamp, onSubmitAnswer, isNewest = false }) => {
  const input = toolUseBlock.input;
  const status = deriveStatus(resultBlock);
  const formattedTime = formatTimestamp(timestamp);

  const [submitted, setSubmitted] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const disabled = status !== "pending" || submitted || isSubmitting || !isNewest;

  if (!isPermissionRequestInput(input)) return null;

  const handleResponse = async (allow: boolean) => {
    if (!onSubmitAnswer || disabled) return;

    const response = JSON.stringify({
      approved: allow,
      tool_name: input.tool_name,
      key: input.key,
    });

    try {
      setIsSubmitting(true);
      await onSubmitAnswer(response);
      setSubmitted(true);
    } finally {
      setIsSubmitting(false);
    }
  };

  const config = STATUS_CONFIG[disabled && status !== "pending" ? status : "pending"];
  const resolvedConfig = STATUS_CONFIG[status];
  const activeConfig = disabled ? resolvedConfig : config;
  const Icon = activeConfig.icon;

  return (
    <div className="mb-3">
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0">
          <div
            className={cn(
              "w-8 h-8 rounded-full flex items-center justify-center",
              activeConfig.avatarClass
            )}
          >
            <Icon className="w-4 h-4 text-white" />
          </div>
        </div>

        <div className="flex-1 min-w-0">
          {formattedTime && (
            <div className="text-[10px] text-muted-foreground/60 mb-0.5">
              {formattedTime}
            </div>
          )}

          <div
            className={cn("rounded-lg border-l-3 pl-3 pr-3 py-2.5", activeConfig.borderClass)}
          >
            <p className="text-sm font-medium text-foreground mb-1">
              Permission Required
            </p>
            <p className="text-sm text-foreground/80 mb-2">
              {input.description}
            </p>

            {(input.file_path || input.command) && (
              <div className="text-xs text-muted-foreground font-mono bg-muted/50 rounded px-2 py-1 mb-2 break-all">
                {input.file_path || input.command}
              </div>
            )}

            {disabled && status !== "pending" && (
              <p className="text-xs text-muted-foreground">
                {status === "approved" ? "Approved" : "Denied"}
              </p>
            )}

            {!disabled && (
              <div className="flex items-center gap-2 mt-2 pt-1.5 border-t border-border/40">
                <Button
                  size="sm"
                  className="h-7 text-xs gap-1 px-3 bg-green-600 hover:bg-green-700 text-white"
                  onClick={() => handleResponse(true)}
                  disabled={isSubmitting}
                >
                  <ShieldCheck className="w-3 h-3" />
                  Allow
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  className="h-7 text-xs gap-1 px-3 text-red-600 border-red-200 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950/30"
                  onClick={() => handleResponse(false)}
                  disabled={isSubmitting}
                >
                  <ShieldX className="w-3 h-3" />
                  Deny
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

PermissionRequestMessage.displayName = "PermissionRequestMessage";
