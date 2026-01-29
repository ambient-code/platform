"use client";

import React, { useState, useMemo } from "react";
import { ThumbsUp, ThumbsDown, Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { FeedbackModal, FeedbackType } from "./FeedbackModal";
import { useFeedbackContextOptional } from "@/contexts/FeedbackContext";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

type FeedbackButtonsProps = {
  messageId?: string;  // Message ID for feedback association
  messageContent?: string;
  messageTimestamp?: string;
  className?: string;
};

export function FeedbackButtons({
  messageId,
  messageContent,
  messageTimestamp,
  className,
}: FeedbackButtonsProps) {
  const [feedbackModalOpen, setFeedbackModalOpen] = useState(false);
  const [selectedFeedback, setSelectedFeedback] = useState<FeedbackType | null>(null);
  const [localSubmittedFeedback, setLocalSubmittedFeedback] = useState<FeedbackType | null>(null);
  
  const feedbackContext = useFeedbackContextOptional();
  
  // Check if this message already has feedback from context (e.g., from replayed META events)
  const existingFeedback = useMemo(() => {
    if (!messageId || !feedbackContext?.messageFeedback) return null;
    const feedback = feedbackContext.messageFeedback.get(messageId);
    if (feedback === 'thumbs_up') return 'positive' as FeedbackType;
    if (feedback === 'thumbs_down') return 'negative' as FeedbackType;
    return null;
  }, [messageId, feedbackContext?.messageFeedback]);
  
  // Use existing feedback from context OR local submission state
  const submittedFeedback = existingFeedback ?? localSubmittedFeedback;
  
  // Don't render if no context available
  if (!feedbackContext) {
    return null;
  }

  const handleFeedbackClick = (type: FeedbackType) => {
    // If already submitted this feedback type, do nothing
    if (submittedFeedback === type) {
      return;
    }
    
    setSelectedFeedback(type);
    setFeedbackModalOpen(true);
  };

  const handleSubmitSuccess = () => {
    setLocalSubmittedFeedback(selectedFeedback);
  };

  const isPositiveSubmitted = submittedFeedback === "positive";
  const isNegativeSubmitted = submittedFeedback === "negative";

  return (
    <>
      <div className={cn("flex items-center gap-1", className)}>
        <TooltipProvider delayDuration={300}>
          {/* Thumbs Up Button */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={() => handleFeedbackClick("positive")}
                disabled={isPositiveSubmitted}
                className={cn(
                  "p-1.5 rounded-md transition-all duration-200",
                  "hover:bg-green-500/10 focus:outline-none focus:ring-2 focus:ring-green-500/30",
                  isPositiveSubmitted
                    ? "text-green-500 bg-green-500/10 cursor-default"
                    : "text-muted-foreground hover:text-green-500 cursor-pointer"
                )}
                aria-label={isPositiveSubmitted ? "Positive feedback submitted" : "This response was helpful"}
              >
                {isPositiveSubmitted ? (
                  <div className="flex items-center gap-1">
                    <ThumbsUp className="h-3.5 w-3.5 fill-current" />
                    <Check className="h-3 w-3" />
                  </div>
                ) : (
                  <ThumbsUp className="h-3.5 w-3.5" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs">
              {isPositiveSubmitted ? "Thanks for your feedback!" : "This was helpful"}
            </TooltipContent>
          </Tooltip>

          {/* Thumbs Down Button */}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={() => handleFeedbackClick("negative")}
                disabled={isNegativeSubmitted}
                className={cn(
                  "p-1.5 rounded-md transition-all duration-200",
                  "hover:bg-red-500/10 focus:outline-none focus:ring-2 focus:ring-red-500/30",
                  isNegativeSubmitted
                    ? "text-red-500 bg-red-500/10 cursor-default"
                    : "text-muted-foreground hover:text-red-500 cursor-pointer"
                )}
                aria-label={isNegativeSubmitted ? "Negative feedback submitted" : "This response was not helpful"}
              >
                {isNegativeSubmitted ? (
                  <div className="flex items-center gap-1">
                    <ThumbsDown className="h-3.5 w-3.5 fill-current" />
                    <Check className="h-3 w-3" />
                  </div>
                ) : (
                  <ThumbsDown className="h-3.5 w-3.5" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs">
              {isNegativeSubmitted ? "Thanks for your feedback!" : "This wasn't helpful"}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      {/* Feedback Modal */}
      {selectedFeedback && (
        <FeedbackModal
          open={feedbackModalOpen}
          onOpenChange={setFeedbackModalOpen}
          feedbackType={selectedFeedback}
          messageId={messageId}
          messageContent={messageContent}
          messageTimestamp={messageTimestamp}
          onSubmitSuccess={handleSubmitSuccess}
        />
      )}
    </>
  );
}
