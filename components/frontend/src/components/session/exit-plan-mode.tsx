"use client";

import React, { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ClipboardCheck, CheckCircle2, Send } from "lucide-react";
import { formatTimestamp } from "@/lib/format-timestamp";
import { hasToolResult } from "@/lib/hitl-tools";
import type { ToolUseBlock, ToolResultBlock } from "@/types/agentic-session";

export type ExitPlanModeMessageProps = {
  toolUseBlock: ToolUseBlock;
  resultBlock?: ToolResultBlock;
  timestamp?: string;
  onSubmitAnswer?: (formattedAnswer: string) => Promise<void>;
  isNewest?: boolean;
};

type AllowedPrompt = {
  tool: string;
  prompt: string;
};

export const ExitPlanModeMessage: React.FC<ExitPlanModeMessageProps> = ({
  toolUseBlock,
  resultBlock,
  timestamp,
  onSubmitAnswer,
  isNewest = false,
}) => {
  const input = toolUseBlock.input;
  const planContent = (input.planContent as string) || "";
  const allowedPrompts = (input.allowedPrompts as AllowedPrompt[]) || [];
  const alreadyAnswered = hasToolResult(resultBlock);
  const formattedTime = formatTimestamp(timestamp);

  const [submitted, setSubmitted] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showFeedback, setShowFeedback] = useState(false);
  const [feedback, setFeedback] = useState("");
  const disabled = alreadyAnswered || submitted || isSubmitting || !isNewest;

  const handleDecision = async (decision: "approve" | "reject" | "request_changes", feedbackText?: string) => {
    if (!onSubmitAnswer || disabled) return;
    const response: Record<string, string> = { decision };
    if (feedbackText) {
      response.feedback = feedbackText;
    }
    try {
      setIsSubmitting(true);
      await onSubmitAnswer(JSON.stringify(response));
      setSubmitted(true);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mb-3">
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <div className="flex-shrink-0">
          <div
            className={cn(
              "w-8 h-8 rounded-full flex items-center justify-center",
              disabled ? "bg-green-600" : "bg-blue-500"
            )}
          >
            {disabled ? (
              <CheckCircle2 className="w-4 h-4 text-white" />
            ) : (
              <ClipboardCheck className="w-4 h-4 text-white" />
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {formattedTime && (
            <div className="text-[10px] text-muted-foreground/60 mb-0.5">{formattedTime}</div>
          )}

          <div
            className={cn(
              "rounded-lg border-l-3 pl-3 pr-3 py-2.5",
              disabled
                ? "border-l-green-500 bg-green-50/30 dark:bg-green-950/10"
                : "border-l-blue-500 bg-blue-50/30 dark:bg-blue-950/10"
            )}
          >
            <p className="text-sm font-medium text-foreground mb-2">Plan Review</p>

            {/* Plan content */}
            {planContent && (
              <div className="text-sm prose prose-sm dark:prose-invert max-w-none mb-3 max-h-96 overflow-y-auto rounded border border-border/40 p-2.5 bg-background/50">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                  {planContent}
                </ReactMarkdown>
              </div>
            )}

            {/* Allowed prompts */}
            {allowedPrompts.length > 0 && (
              <div className="mb-3">
                <p className="text-xs font-medium text-muted-foreground mb-1">Requested permissions:</p>
                <ul className="space-y-0.5">
                  {allowedPrompts.map((p) => (
                    <li key={`${p.tool}:${p.prompt}`} className="text-xs text-muted-foreground flex items-center gap-1.5">
                      <span className="inline-block w-1 h-1 rounded-full bg-muted-foreground/50 flex-shrink-0" />
                      <span className="font-mono">{p.tool}</span>: {p.prompt}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* Request changes feedback input */}
            {showFeedback && !disabled && (
              <div className="mb-2">
                <Input
                  autoFocus
                  placeholder="Describe the changes you'd like..."
                  value={feedback}
                  onChange={(e) => setFeedback(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey && feedback.trim()) {
                      e.preventDefault();
                      handleDecision("request_changes", feedback.trim());
                    }
                  }}
                  disabled={disabled}
                  className="h-8 text-sm"
                />
              </div>
            )}

            {/* Action buttons */}
            {!disabled && (
              <div className="flex items-center gap-1.5 mt-2 pt-1.5 border-t border-border/40">
                {!showFeedback ? (
                  <>
                    <Button
                      size="sm"
                      className="h-7 text-xs gap-1 px-3"
                      onClick={() => handleDecision("approve")}
                    >
                      <CheckCircle2 className="w-3 h-3" />
                      Approve
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs px-3"
                      onClick={() => handleDecision("reject")}
                    >
                      Reject
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs px-3"
                      onClick={() => setShowFeedback(true)}
                    >
                      Request Changes
                    </Button>
                  </>
                ) : (
                  <>
                    <Button
                      size="sm"
                      className="h-7 text-xs gap-1 px-3"
                      onClick={() => handleDecision("request_changes", feedback.trim())}
                      disabled={!feedback.trim()}
                    >
                      <Send className="w-3 h-3" />
                      Send Feedback
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs px-3"
                      onClick={() => setShowFeedback(false)}
                    >
                      Cancel
                    </Button>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

ExitPlanModeMessage.displayName = "ExitPlanModeMessage";
