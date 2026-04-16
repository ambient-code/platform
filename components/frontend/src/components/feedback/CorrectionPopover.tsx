"use client";

import React, { useState, useCallback } from "react";
import { PencilLine, Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useFeedbackContextOptional } from "@/contexts/FeedbackContext";
import type { CorrectionType } from "@/services/api/corrections";
import { submitCorrection, CORRECTION_TYPE_LABELS } from "@/services/api/corrections";

const MIN_CORRECTION_LENGTH = 10;
const MAX_CORRECTION_LENGTH = 2000;
const SUCCESS_COOLDOWN_MS = 2000;

type CorrectionPopoverProps = {
  messageId?: string;
  messageContent?: string;
  className?: string;
};

export function CorrectionPopover({
  messageId,
  messageContent,
  className,
}: CorrectionPopoverProps) {
  const [open, setOpen] = useState(false);
  const [correctionType, setCorrectionType] = useState<CorrectionType | "">("");
  const [correctionText, setCorrectionText] = useState("");
  const [includeContent, setIncludeContent] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submittedCount, setSubmittedCount] = useState(0);
  const [submitCooldown, setSubmitCooldown] = useState(false);

  const feedbackContext = useFeedbackContextOptional();

  const charCount = correctionText.length;
  const isValid =
    correctionType !== "" &&
    charCount >= MIN_CORRECTION_LENGTH &&
    charCount <= MAX_CORRECTION_LENGTH;
  const canSubmit = isValid && !isSubmitting && !submitCooldown;

  const resetForm = useCallback(() => {
    setCorrectionType("");
    setCorrectionText("");
    setIncludeContent(false);
    setError(null);
  }, []);

  const handleSubmit = async () => {
    if (!canSubmit || !feedbackContext || !messageId) return;

    setIsSubmitting(true);
    setError(null);

    try {
      await submitCorrection(feedbackContext.projectName, {
        correction_type: correctionType as CorrectionType,
        user_correction: correctionText,
        session_name: feedbackContext.sessionName,
        message_id: messageId,
        message_content: includeContent ? messageContent : undefined,
        source: "ui",
      });

      setSubmittedCount((prev) => prev + 1);
      resetForm();
      setOpen(false);

      // Prevent rapid re-submission
      setSubmitCooldown(true);
      setTimeout(() => setSubmitCooldown(false), SUCCESS_COOLDOWN_MS);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to submit correction"
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    resetForm();
    setOpen(false);
  };

  // Don't render if no context available
  if (!feedbackContext) {
    return null;
  }

  const hasSubmitted = submittedCount > 0;

  return (
    <TooltipProvider delayDuration={300}>
      <Popover open={open} onOpenChange={setOpen}>
        <Tooltip>
          <TooltipTrigger asChild>
            <PopoverTrigger asChild>
              <button
                className={cn(
                  "p-1.5 rounded-md transition-all duration-200",
                  "hover:bg-amber-500/10 focus:outline-none focus:ring-2 focus:ring-amber-500/30",
                  hasSubmitted
                    ? "text-amber-500 bg-amber-500/10"
                    : "text-muted-foreground hover:text-amber-500 cursor-pointer",
                  className
                )}
                aria-label={
                  hasSubmitted
                    ? `Correction submitted (${submittedCount})`
                    : "Correct this response"
                }
              >
                <div className="flex items-center gap-1">
                  <PencilLine className="h-3.5 w-3.5" />
                  {hasSubmitted && <Check className="h-3 w-3" />}
                </div>
              </button>
            </PopoverTrigger>
          </TooltipTrigger>
          <TooltipContent side="top" className="text-xs">
            {hasSubmitted ? "Correction submitted" : "Correct this"}
          </TooltipContent>
        </Tooltip>

        <PopoverContent
          align="start"
          side="top"
          className="w-80"
          onInteractOutside={(e) => {
            // Prevent closing when submitting (Select dropdown renders in a portal)
            if (isSubmitting) {
              e.preventDefault();
            }
          }}
        >
          <div className="space-y-3">
            <div className="text-sm font-medium">Correct this response</div>

            {/* Correction type */}
            <div className="space-y-1.5">
              <Label htmlFor="correction-type" className="text-xs">
                Correction type
              </Label>
              <Select
                value={correctionType}
                onValueChange={(val) =>
                  setCorrectionType(val as CorrectionType)
                }
              >
                <SelectTrigger
                  id="correction-type"
                  className="w-full h-8 text-xs"
                >
                  <SelectValue placeholder="Select type..." />
                </SelectTrigger>
                <SelectContent>
                  {(
                    Object.entries(CORRECTION_TYPE_LABELS) as Array<
                      [CorrectionType, string]
                    >
                  ).map(([value, label]) => (
                    <SelectItem key={value} value={value} className="text-xs">
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Free-text correction */}
            <div className="space-y-1.5">
              <Label htmlFor="correction-text" className="text-xs">
                What should have happened instead?
              </Label>
              <Textarea
                id="correction-text"
                placeholder="Describe the correct behavior..."
                value={correctionText}
                onChange={(e) =>
                  setCorrectionText(
                    e.target.value.slice(0, MAX_CORRECTION_LENGTH)
                  )
                }
                rows={3}
                className="resize-none text-xs"
              />
              <div
                className={cn(
                  "text-[10px] text-right",
                  charCount > MAX_CORRECTION_LENGTH
                    ? "text-destructive"
                    : charCount < MIN_CORRECTION_LENGTH && charCount > 0
                      ? "text-amber-500"
                      : "text-muted-foreground"
                )}
              >
                {charCount}/{MAX_CORRECTION_LENGTH}
              </div>
            </div>

            {/* Include message content checkbox */}
            <div className="flex items-center gap-2">
              <Checkbox
                id="include-content"
                checked={includeContent}
                onCheckedChange={(checked) =>
                  setIncludeContent(checked === true)
                }
              />
              <Label
                htmlFor="include-content"
                className="text-xs text-muted-foreground cursor-pointer"
              >
                Include message content as context
              </Label>
            </div>

            {/* Error message */}
            {error && (
              <div className="rounded-md bg-destructive/10 p-2 text-xs text-destructive">
                {error}
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end gap-2 pt-1">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCancel}
                disabled={isSubmitting}
                className="h-7 text-xs"
              >
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleSubmit}
                disabled={!canSubmit}
                className="h-7 text-xs"
              >
                {isSubmitting && (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                )}
                Submit
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  );
}
