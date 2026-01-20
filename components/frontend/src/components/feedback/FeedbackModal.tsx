"use client";

import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { ThumbsUp, ThumbsDown, Loader2, Info } from "lucide-react";
import { useFeedbackContextOptional } from "@/contexts/FeedbackContext";
import type { MessageObject, ToolUseMessages } from "@/types/agentic-session";

export type FeedbackType = "positive" | "negative";

type FeedbackModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  feedbackType: FeedbackType;
  messageContent?: string;
  messageTimestamp?: string;
  onSubmitSuccess?: () => void;
};

// Helper to extract text content from messages
function extractMessageText(
  messages: Array<MessageObject | ToolUseMessages>
): Array<{ role: string; content: string; timestamp?: string }> {
  return messages
    .filter((m): m is MessageObject => "type" in m && m.type !== undefined)
    .filter((m) => m.type === "user_message" || m.type === "agent_message")
    .map((m) => {
      let content = "";
      if (typeof m.content === "string") {
        content = m.content;
      } else if ("text" in m.content) {
        content = m.content.text;
      } else if ("thinking" in m.content) {
        content = m.content.thinking;
      }
      return {
        role: m.type === "user_message" ? "user" : "assistant",
        content,
        timestamp: m.timestamp,
      };
    });
}

export function FeedbackModal({
  open,
  onOpenChange,
  feedbackType,
  messageContent,
  onSubmitSuccess,
}: FeedbackModalProps) {
  const [comment, setComment] = useState("");
  const [includeTranscript, setIncludeTranscript] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const feedbackContext = useFeedbackContextOptional();

  const handleSubmit = async () => {
    if (!feedbackContext) {
      setError("Session context not available");
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      // Build context string from what the user was working on
      const contextParts: string[] = [];
      
      if (feedbackContext.initialPrompt) {
        contextParts.push(`Initial prompt: ${feedbackContext.initialPrompt}`);
      }
      
      if (messageContent) {
        contextParts.push(messageContent);
      }

      // Extract all messages for the transcript
      const transcript = includeTranscript
        ? extractMessageText(feedbackContext.messages)
        : undefined;

      // Build AG-UI META event following the spec
      // See: https://docs.ag-ui.com/drafts/meta-events#user-feedback
      const payload: Record<string, unknown> = {
        userId: feedbackContext.username,
        projectName: feedbackContext.projectName,
        sessionName: feedbackContext.sessionName,
      };
      
      if (feedbackContext.traceId) {
        payload.traceId = feedbackContext.traceId;
      }
      if (comment) {
        payload.comment = comment;
      }
      if (feedbackContext.activeWorkflow) {
        payload.workflow = feedbackContext.activeWorkflow;
      }
      if (contextParts.length > 0) {
        payload.context = contextParts.join("; ");
      }
      if (includeTranscript && transcript && transcript.length > 0) {
        payload.includeTranscript = true;
        payload.transcript = transcript;
      }
      
      const metaEvent = {
        type: "META",
        metaType: feedbackType === "positive" ? "thumbs_up" : "thumbs_down",
        payload,
        threadId: feedbackContext.sessionName,
        ts: Date.now(),
      };
      
      // Send to backend (which forwards to runner and broadcasts on event stream)
      const feedbackUrl = `/api/projects/${encodeURIComponent(feedbackContext.projectName)}/agentic-sessions/${encodeURIComponent(feedbackContext.sessionName)}/agui/feedback`;
      
      const response = await fetch(feedbackUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(metaEvent),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to submit feedback");
      }

      // Success - close modal and reset
      setComment("");
      setIncludeTranscript(false);
      onOpenChange(false);
      onSubmitSuccess?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit feedback");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    setComment("");
    setIncludeTranscript(false);
    setError(null);
    onOpenChange(false);
  };

  const isPositive = feedbackType === "positive";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {isPositive ? (
              <ThumbsUp className="h-5 w-5 text-green-500" />
            ) : (
              <ThumbsDown className="h-5 w-5 text-red-500" />
            )}
            <span>Share feedback</span>
          </DialogTitle>
          <DialogDescription>
            {isPositive
              ? "Help us improve by sharing what went well."
              : "Help us improve by sharing what went wrong."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Comment textarea */}
          <div className="space-y-2">
            <Label htmlFor="feedback-comment">
              Additional comments (optional)
            </Label>
            <Textarea
              id="feedback-comment"
              placeholder={
                isPositive
                  ? "What was good about this response?"
                  : "What could be improved about this response?"
              }
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
              className="resize-none"
            />
          </div>

          {/* Include transcript checkbox */}
          <div className="flex items-start space-x-3">
            <Checkbox
              id="include-transcript"
              checked={includeTranscript}
              onCheckedChange={(checked) => setIncludeTranscript(checked === true)}
            />
            <div className="space-y-1">
              <Label
                htmlFor="include-transcript"
                className="text-sm font-medium cursor-pointer"
              >
                Include all messages
              </Label>
              <p className="text-xs text-muted-foreground">
                This helps us understand the full context of your experience.
              </p>
            </div>
          </div>

          {/* Privacy disclaimer */}
          <div className="rounded-md border border-border/50 bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5 mb-1">
              <Info className="h-3.5 w-3.5 flex-shrink-0" />
              <span className="font-medium">Privacy</span>
            </div>
            <p>
              {includeTranscript
                ? "Your feedback and all messages in this session will be stored to help improve the platform."
                : "Your feedback and this message will be stored to help improve the platform."}
            </p>
          </div>

          {/* Error message */}
          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={handleCancel} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Send feedback
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
