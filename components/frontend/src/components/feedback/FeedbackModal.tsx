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

export type FeedbackType = "positive" | "negative";

type FeedbackModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  feedbackType: FeedbackType;
  projectName: string;
  sessionName: string;
  messageId?: string;
  messageContent?: string;
  /** AG-UI runId from the assistant message — used by backend to resolve the Langfuse traceId. */
  runId?: string;
  onSubmitSuccess?: () => void;
};

/**
 * Self-contained feedback modal that sends a META event to the
 * platform's AG-UI feedback endpoint.  The backend forwards to
 * the runner (Langfuse) and persists a RAW event so the thumbs
 * highlight survives reconnects.  No dependency on FeedbackContext.
 *
 * Used by CopilotChatPanel when CopilotKit's built-in thumbs up/down
 * buttons are clicked.
 */
export function FeedbackModal({
  open,
  onOpenChange,
  feedbackType,
  projectName,
  sessionName,
  messageId,
  messageContent,
  runId,
  onSubmitSuccess,
}: FeedbackModalProps) {
  const [comment, setComment] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setIsSubmitting(true);
    setError(null);

    try {
      const payload: Record<string, unknown> = {
        projectName,
        sessionName,
      };

      if (messageId) {
        payload.messageId = messageId;
      }
      if (runId) {
        payload.runId = runId;
      }
      if (comment) {
        payload.comment = comment;
      }
      if (messageContent) {
        payload.context = messageContent;
      }

      const metaEvent = {
        type: "META",
        metaType: feedbackType === "positive" ? "thumbs_up" : "thumbs_down",
        payload,
        threadId: sessionName,
        ts: Date.now(),
      };

      const feedbackUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/feedback`;

      const response = await fetch(feedbackUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(metaEvent),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to submit feedback");
      }

      // Success
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

          <div className="rounded-md border border-border/50 bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5 mb-1">
              <Info className="h-3.5 w-3.5 flex-shrink-0" />
              <span className="font-medium">Privacy</span>
            </div>
            <p>
              Your feedback and this message will be stored to help improve the platform.
            </p>
          </div>

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
