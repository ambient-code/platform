"use client";

import React from "react";
import { Clock, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { QueuedMessageItem } from "@/hooks/use-session-queue";

export type QueuedMessageBubbleProps = {
  message: QueuedMessageItem;
  onCancel: (messageId: string) => void;
};

export const QueuedMessageBubble: React.FC<QueuedMessageBubbleProps> = ({
  message,
  onCancel,
}) => {
  const timeAgo = React.useMemo(() => {
    const seconds = Math.floor((Date.now() - message.timestamp) / 1000);
    if (seconds < 60) return "just now";
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    return `${hours}h ago`;
  }, [message.timestamp]);

  return (
    <div className="mb-4 mt-2">
      <div className="flex space-x-3 items-start">
        {/* User Avatar */}
        <div className="flex-shrink-0">
          <div className="w-8 h-8 rounded-full flex items-center justify-center bg-muted">
            <span className="text-muted-foreground text-sm font-semibold">You</span>
          </div>
        </div>

        {/* Message Content */}
        <div className="flex-1 min-w-0">
          {/* Timestamp */}
          <div className="text-sm text-muted-foreground/60 mb-1">{timeAgo}</div>

          {/* Queued message with distinct styling */}
          <div className="bg-status-warning border-l-2 border-status-warning-border rounded-r-md p-3">
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-status-warning-foreground" />
                <span className="text-sm font-medium text-status-warning-foreground">
                  Queued
                </span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onCancel(message.id)}
                className="h-6 px-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
              >
                <X className="h-3 w-3 mr-1" />
                Cancel
              </Button>
            </div>
            <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default QueuedMessageBubble;
