"use client";

import { useMemo } from "react";
import { Workflow, ChevronDown, Loader2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import { cn } from "@/lib/utils";
import { useWorkflowSelection } from "../hooks/use-workflow-selection";
import type { WorkflowConfig } from "../lib/types";

type ActiveWorkflowDetails = {
  gitUrl: string;
  branch: string;
  path: string;
};

type WorkflowSelectorProps = {
  sessionPhase?: string;
  activeWorkflow: string | null;
  activeWorkflowDetails?: ActiveWorkflowDetails;
  selectedWorkflow: string;
  workflowActivating: boolean;
  ootbWorkflows: WorkflowConfig[];
  onWorkflowChange: (value: string) => void;
  onLoadCustom?: () => void;
};

export function WorkflowSelector({
  sessionPhase,
  activeWorkflow,
  activeWorkflowDetails,
  selectedWorkflow,
  workflowActivating,
  ootbWorkflows,
  onWorkflowChange,
  onLoadCustom,
}: WorkflowSelectorProps) {
  const isSessionStopped =
    sessionPhase === "Stopped" ||
    sessionPhase === "Error" ||
    sessionPhase === "Completed";

  const {
    search,
    setSearch,
    popoverOpen,
    searchInputRef,
    filteredWorkflows,
    showGeneralChat,
    selectedLabel,
    isActivating,
    handleSelect,
    handleOpenChange,
  } = useWorkflowSelection({
    selectedWorkflow,
    ootbWorkflows,
    workflowActivating,
    onWorkflowChange,
  });

  // Determine display label — show active workflow name or "No workflow"
  const displayLabel = useMemo(() => {
    if (!activeWorkflow) return selectedLabel;
    return ootbWorkflows.find((w) => w.id === activeWorkflow)?.name || "Custom workflow";
  }, [activeWorkflow, ootbWorkflows, selectedLabel]);

  const isCustomWithDetails = activeWorkflow === "custom" && activeWorkflowDetails;

  const triggerButton = (
    <Button
      variant="ghost"
      size="sm"
      disabled={isSessionStopped || isActivating}
      className="gap-1.5 text-muted-foreground hover:text-foreground"
    >
      {isActivating ? (
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
      ) : (
        <Workflow className="h-3.5 w-3.5" />
      )}
      <span className="max-w-[140px] truncate text-sm">
        {isActivating ? "Switching..." : displayLabel}
      </span>
      <ChevronDown className="h-3 w-3 opacity-50" />
    </Button>
  );

  const popoverTrigger = (
    <PopoverTrigger asChild>
      {triggerButton}
    </PopoverTrigger>
  );

  return (
    <Popover open={popoverOpen} onOpenChange={handleOpenChange}>
      {isCustomWithDetails ? (
        <TooltipProvider delayDuration={300}>
          <Tooltip>
            <TooltipTrigger asChild>
              {popoverTrigger}
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[350px]">
              <div className="space-y-1 text-xs">
                <div className="truncate"><span className="text-muted-foreground">Repo:</span> {activeWorkflowDetails.gitUrl}</div>
                <div><span className="text-muted-foreground">Branch:</span> {activeWorkflowDetails.branch}</div>
                {activeWorkflowDetails.path && (
                  <div className="truncate"><span className="text-muted-foreground">Path:</span> {activeWorkflowDetails.path}</div>
                )}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ) : (
        popoverTrigger
      )}
      <PopoverContent
        className="w-[350px] p-0"
        align="end"
        side="top"
        sideOffset={8}
      >
        {/* Search */}
        <div className="px-2 py-2 border-b sticky top-0 bg-popover z-10">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchInputRef}
              type="text"
              placeholder="Search workflows..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 h-8"
              onKeyDown={(e) => e.stopPropagation()}
            />
          </div>
        </div>

        {/* Workflow list */}
        <div className="max-h-[300px] overflow-y-auto">
          {showGeneralChat && (
            <>
              <WorkflowItem
                name="General chat"
                description="A general chat session with no structured workflow."
                selected={selectedWorkflow === "none"}
                onClick={() => handleSelect("none")}
              />
              {filteredWorkflows.length > 0 && <div className="border-t my-1" />}
            </>
          )}
          {filteredWorkflows.map((workflow) => (
            <WorkflowItem
              key={workflow.id}
              name={workflow.name}
              description={workflow.description}
              selected={selectedWorkflow === workflow.id}
              disabled={!workflow.enabled}
              onClick={() => workflow.enabled && handleSelect(workflow.id)}
            />
          ))}
          {!showGeneralChat &&
            filteredWorkflows.length === 0 && (
              <div className="px-2 py-6 text-center text-sm text-muted-foreground">
                No workflows found
              </div>
            )}
        </div>

        {/* Footer */}
        {onLoadCustom && (
          <div className="border-t px-3 py-2 flex items-center justify-between text-xs text-muted-foreground">
            <span>
              {ootbWorkflows.length} workflow
              {ootbWorkflows.length === 1 ? "" : "s"} available
            </span>
            <button
              type="button"
              className="text-primary hover:underline"
              onClick={() => {
                handleSelect("custom");
                onLoadCustom();
              }}
            >
              Load custom
            </button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

function WorkflowItem({
  name,
  description,
  selected,
  disabled,
  onClick,
}: {
  name: string;
  description: string;
  selected: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "w-full text-left px-3 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer",
        selected && "bg-accent",
        disabled && "opacity-50 cursor-not-allowed"
      )}
    >
      <div className="flex flex-col items-start gap-0.5 py-0.5">
        <span className="text-sm">{name}</span>
        <span className="text-xs text-muted-foreground font-normal line-clamp-2">
          {description}
        </span>
      </div>
    </button>
  );
}
