"use client";

import { useState, useRef, useMemo, useEffect } from "react";
import { Search, ChevronDown, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { WorkflowConfig } from "@/types/workflow";

type WorkflowPickerProps = {
  selectedWorkflow: string;
  ootbWorkflows: WorkflowConfig[];
  onWorkflowChange: (value: string) => void;
  disabled?: boolean;
  isLoading?: boolean;
  loadingMessage?: string;
  placeholder?: string;
  className?: string;
  showGeneralChat?: boolean;
  showCustomWorkflow?: boolean;
};

export function WorkflowPicker({
  selectedWorkflow,
  ootbWorkflows,
  onWorkflowChange,
  disabled = false,
  isLoading = false,
  loadingMessage,
  placeholder,
  className,
  showGeneralChat = true,
  showCustomWorkflow = true,
}: WorkflowPickerProps) {
  const [workflowSearch, setWorkflowSearch] = useState("");
  const [popoverOpen, setPopoverOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const focusTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const filteredWorkflows = useMemo(
    () =>
      ootbWorkflows
        .filter((workflow) => {
          if (!workflowSearch) return true;
          const searchLower = workflowSearch.toLowerCase();
          return (
            workflow.name.toLowerCase().includes(searchLower) ||
            workflow.description.toLowerCase().includes(searchLower)
          );
        })
        .sort((a, b) => a.name.localeCompare(b.name)),
    [ootbWorkflows, workflowSearch]
  );

  const showGeneralChatOption = useMemo(
    () =>
      showGeneralChat &&
      (!workflowSearch ||
        "general chat".includes(workflowSearch.toLowerCase()) ||
        "no structured workflow".includes(workflowSearch.toLowerCase())),
    [showGeneralChat, workflowSearch]
  );

  const showCustomWorkflowOption = useMemo(
    () =>
      showCustomWorkflow &&
      (!workflowSearch ||
        "custom".includes(workflowSearch.toLowerCase()) ||
        "git repository".includes(workflowSearch.toLowerCase())),
    [showCustomWorkflow, workflowSearch]
  );

  const selectedWorkflowInfo = useMemo(() => {
    if (selectedWorkflow === "none") {
      return {
        name: "General chat",
        description: "A general chat session with no structured workflow."
      };
    }
    if (selectedWorkflow === "custom") {
      return {
        name: "Custom workflow...",
        description: "Load a workflow from a custom Git repository"
      };
    }
    const workflow = ootbWorkflows.find(w => w.id === selectedWorkflow);
    return workflow
      ? { name: workflow.name, description: workflow.description }
      : { name: placeholder || "Select workflow...", description: "" };
  }, [selectedWorkflow, ootbWorkflows, placeholder]);

  const handleWorkflowSelect = (value: string) => {
    onWorkflowChange(value);
    setPopoverOpen(false);
  };

  useEffect(() => {
    return () => {
      if (focusTimeoutRef.current) {
        clearTimeout(focusTimeoutRef.current);
      }
    };
  }, []);

  if (isLoading) {
    return (
      <Button
        type="button"
        variant="outline"
        className={cn("w-full h-auto py-3 justify-between", className)}
        disabled
      >
        <div className="flex flex-col items-start gap-0.5 w-full">
          <div className="flex items-center gap-2">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            <span>{loadingMessage || "Loading workflows..."}</span>
          </div>
          <span className="text-xs text-muted-foreground font-normal">
            This may take a few seconds...
          </span>
        </div>
      </Button>
    );
  }

  return (
    <Popover open={popoverOpen} onOpenChange={(open) => {
      setPopoverOpen(open);
      if (open) {
        setWorkflowSearch("");
        if (focusTimeoutRef.current) {
          clearTimeout(focusTimeoutRef.current);
        }
        focusTimeoutRef.current = setTimeout(() => {
          searchInputRef.current?.focus();
          focusTimeoutRef.current = null;
        }, 0);
      } else {
        if (focusTimeoutRef.current) {
          clearTimeout(focusTimeoutRef.current);
          focusTimeoutRef.current = null;
        }
      }
    }}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={popoverOpen}
          aria-label="Select workflow"
          className={cn("w-full h-auto py-3 justify-between", className)}
          disabled={disabled}
        >
          <div className="flex items-start justify-between w-full gap-2">
            <div className="flex flex-col items-start gap-0.5 text-left flex-1 min-w-0">
              <span className="font-medium truncate w-full">{selectedWorkflowInfo.name}</span>
              {selectedWorkflowInfo.description && (
                <span className="text-xs text-muted-foreground font-normal line-clamp-2 w-full">
                  {selectedWorkflowInfo.description}
                </span>
              )}
            </div>
            <ChevronDown className="h-4 w-4 shrink-0 opacity-50 mt-1" />
          </div>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[450px] p-0" align="start" sideOffset={4}>
        <div className="px-2 py-2 border-b sticky top-0 bg-popover z-10">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchInputRef}
              type="text"
              placeholder="Search workflows..."
              value={workflowSearch}
              onChange={(e) => setWorkflowSearch(e.target.value)}
              className="pl-8 h-9"
              aria-label="Search workflows"
              onKeyDown={(e) => {
                e.stopPropagation();
              }}
            />
          </div>
        </div>

        <div className="max-h-[400px] overflow-y-auto">
          {showGeneralChatOption && (
            <>
              <button
                type="button"
                onClick={() => handleWorkflowSelect("none")}
                className={cn(
                  "w-full text-left px-2 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer transition-colors",
                  selectedWorkflow === "none" && "bg-accent"
                )}
                aria-pressed={selectedWorkflow === "none"}
              >
                <div className="flex flex-col items-start gap-0.5 py-1">
                  <span className="text-sm font-medium">General chat</span>
                  <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                    A general chat session with no structured workflow.
                  </span>
                </div>
              </button>
              {filteredWorkflows.length > 0 && <div className="border-t my-1" />}
            </>
          )}
          {filteredWorkflows.map((workflow) => (
            <button
              key={workflow.id}
              type="button"
              onClick={() => workflow.enabled && handleWorkflowSelect(workflow.id)}
              disabled={!workflow.enabled}
              className={cn(
                "w-full text-left px-2 py-2 transition-colors",
                workflow.enabled && "hover:bg-accent hover:text-accent-foreground cursor-pointer",
                selectedWorkflow === workflow.id && "bg-accent",
                !workflow.enabled && "opacity-50 cursor-not-allowed"
              )}
              aria-pressed={selectedWorkflow === workflow.id}
            >
              <div className="flex flex-col items-start gap-0.5 py-1">
                <span className="text-sm font-medium">{workflow.name}</span>
                <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                  {workflow.description}
                </span>
              </div>
            </button>
          ))}
          {(showGeneralChatOption || filteredWorkflows.length > 0) && showCustomWorkflowOption && (
            <div className="border-t my-1" />
          )}
          {showCustomWorkflowOption && (
            <button
              type="button"
              onClick={() => handleWorkflowSelect("custom")}
              className={cn(
                "w-full text-left px-2 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer transition-colors",
                selectedWorkflow === "custom" && "bg-accent"
              )}
              aria-pressed={selectedWorkflow === "custom"}
            >
              <div className="flex flex-col items-start gap-0.5 py-1">
                <span className="text-sm font-medium">Custom workflow...</span>
                <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                  Load a workflow from a custom Git repository
                </span>
              </div>
            </button>
          )}
          {!showGeneralChatOption && filteredWorkflows.length === 0 && !showCustomWorkflowOption && (
            <div className="px-2 py-6 text-center text-sm text-muted-foreground">
              No workflows found
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
