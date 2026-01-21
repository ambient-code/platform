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
import { Label } from "@/components/ui/label";
import { ThumbsUp, ThumbsDown, Loader2, Info } from "lucide-react";
import { useFeedbackContextOptional } from "@/contexts/FeedbackContext";

export type FeedbackType = "positive" | "negative";

type FeedbackModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  feedbackType: FeedbackType;
  messageId?: string;  // Message ID for feedback association (matches messages in MESSAGES_SNAPSHOT)
  messageContent?: string;
  messageTimestamp?: string;
  onSubmitSuccess?: () => void;
};

export function FeedbackModal({
  open,
  onOpenChange,
  feedbackType,
  messageId,
  messageContent,
  onSubmitSuccess,
}: FeedbackModalProps) {
  const [comment, setComment] = useState("");
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

      // Build AG-UI META event following the spec
      // See: https://docs.ag-ui.com/drafts/meta-events#user-feedback
      const payload: Record<string, unknown> = {
        userId: feedbackContext.username,
        projectName: feedbackContext.projectName,
        sessionName: feedbackContext.sessionName,
      };
      
      // Include messageId so frontend can match feedback to specific messages
      if (messageId) {
        payload.messageId = messageId;
      }
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

          {/* Privacy disclaimer */}
          <div className="rounded-md border border-border/50 bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5 mb-1">
              <Info className="h-3.5 w-3.5 flex-shrink-0" />
              <span className="font-medium">Privacy</span>
            </div>
            <p>
              Your feedback and this message will be stored to help improve the platform.
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
