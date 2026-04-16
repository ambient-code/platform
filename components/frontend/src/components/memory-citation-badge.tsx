"use client";

import React from "react";
import { cn } from "@/lib/utils";
import { BookOpen, AlertTriangle } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

type MemoryCitationBadgeProps = {
  memoryId: string;
  summary?: string;
  fullContent?: string;
  author?: string;
  createdAt?: string;
  correctionId?: string;
  deprecated?: boolean;
  notFound?: boolean;
};

/**
 * Inline badge component for memory citations.
 * Renders [memory:PM-XXX] patterns as styled pill/chip with popover details.
 */
export function MemoryCitationBadge({
  memoryId,
  summary,
  fullContent,
  author,
  createdAt,
  correctionId,
  deprecated = false,
  notFound = false,
}: MemoryCitationBadgeProps) {
  if (notFound) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={cn(
                "inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full",
                "text-[11px] font-medium leading-none align-baseline",
                "bg-muted text-muted-foreground border border-border",
                "cursor-default"
              )}
            >
              <AlertTriangle className="w-3 h-3 text-amber-500 flex-shrink-0" />
              <span>{memoryId}</span>
            </span>
          </TooltipTrigger>
          <TooltipContent>Memory not found</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          data-deprecated={deprecated || undefined}
          className={cn(
            "inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full",
            "text-[11px] font-medium leading-none align-baseline",
            "border transition-colors cursor-pointer",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            deprecated
              ? "bg-muted/50 text-muted-foreground/60 border-border/50 line-through"
              : "bg-primary/10 text-primary border-primary/20 hover:bg-primary/20"
          )}
        >
          <BookOpen className="w-3 h-3 flex-shrink-0" />
          <span>{memoryId}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="start">
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="font-semibold text-sm">{memoryId}</span>
            {deprecated && (
              <span className="text-[10px] text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-900/30 px-1.5 py-0.5 rounded">
                Deprecated
              </span>
            )}
          </div>
          {(fullContent || summary) && (
            <p className="text-sm text-muted-foreground leading-relaxed">
              {fullContent || summary}
            </p>
          )}
          {(author || createdAt || correctionId) && (
            <div className="text-[11px] text-muted-foreground/70 space-y-0.5 pt-1 border-t">
              {author && <div>Author: {author}</div>}
              {createdAt && <div>Created: {createdAt}</div>}
              {correctionId && <div>From correction: {correctionId}</div>}
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

type MemoryCitationSummaryProps = {
  count: number;
};

/**
 * Summary chip shown when more than 10 memories are cited in a single message (FR-016).
 */
export function MemoryCitationSummary({ count }: MemoryCitationSummaryProps) {
  if (count <= 10) {
    return null;
  }

  return (
    <div className="inline-flex items-center gap-1.5 px-2 py-1 mb-2 rounded-md bg-primary/5 border border-primary/10 text-xs text-primary">
      <BookOpen className="w-3.5 h-3.5" />
      <span>{count} memories cited</span>
    </div>
  );
}
