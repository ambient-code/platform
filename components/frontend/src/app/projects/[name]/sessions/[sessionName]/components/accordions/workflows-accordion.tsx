"use client";

import { useState, useRef } from "react";
import { Play, Loader2, Workflow, Search, ChevronDown } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { WorkflowConfig } from "../../lib/types";

type WorkflowsAccordionProps = {
  sessionPhase?: string;
  activeWorkflow: string | null;
  selectedWorkflow: string;
  workflowActivating: boolean;
  ootbWorkflows: WorkflowConfig[];
  isExpanded: boolean;
  onWorkflowChange: (value: string) => void;
  onResume?: () => void;
};

export function WorkflowsAccordion({
  sessionPhase,
  activeWorkflow,
  selectedWorkflow,
  workflowActivating,
  ootbWorkflows,
  isExpanded,
  onWorkflowChange,
  onResume,
}: WorkflowsAccordionProps) {
  const [workflowSearch, setWorkflowSearch] = useState("");
  const [popoverOpen, setPopoverOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const isSessionStopped = sessionPhase === 'Stopped' || sessionPhase === 'Error' || sessionPhase === 'Completed';

  // Filter workflows based on search query
  const filteredWorkflows = ootbWorkflows
    .filter((workflow) => {
      if (!workflowSearch) return true;
      const searchLower = workflowSearch.toLowerCase();
      return (
        workflow.name.toLowerCase().includes(searchLower) ||
        workflow.description.toLowerCase().includes(searchLower)
      );
    })
    .sort((a, b) => a.name.localeCompare(b.name)); // Sort alphabetically by display name

  // Filter for general chat based on search
  const showGeneralChat = !workflowSearch || 
    "general chat".includes(workflowSearch.toLowerCase()) ||
    "A general chat session with no structured workflow.".toLowerCase().includes(workflowSearch.toLowerCase());

  // Filter for custom workflow based on search
  const showCustomWorkflow = !workflowSearch ||
    "custom workflow".toLowerCase().includes(workflowSearch.toLowerCase()) ||
    "load a workflow from a custom git repository".toLowerCase().includes(workflowSearch.toLowerCase());

  // Get display info for selected workflow
  const getSelectedWorkflowInfo = () => {
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
      : { name: "Select workflow...", description: "" };
  };

  const handleWorkflowSelect = (value: string) => {
    onWorkflowChange(value);
    setPopoverOpen(false);
  };

  return (
    <AccordionItem value="workflows" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Workflow className="h-4 w-4" />
          <span>Workflows</span>
          {activeWorkflow && !isExpanded && (
            <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800">
              {ootbWorkflows.find(w => w.id === activeWorkflow)?.name || "Custom Workflow"}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="pt-2 pb-3">
        {isSessionStopped ? (
          <div className="py-8 flex flex-col items-center justify-center space-y-4">
            <Play className="h-12 w-12 text-muted-foreground/50" />
            <div className="text-center space-y-1">
              <h3 className="font-medium text-sm">Session not running</h3>
              <p className="text-sm text-muted-foreground">
                You need to resume this session to use workflows.
              </p>
            </div>
            {onResume && sessionPhase === 'Stopped' && (
              <Button
                onClick={onResume}
                size="sm"
                className="hover:border-green-600 hover:bg-green-50 group"
                variant="outline"
              >
                <Play className="w-4 h-4 mr-2 fill-green-200 stroke-green-600 group-hover:fill-green-500 group-hover:stroke-green-700 transition-colors" />
                Resume Session
              </Button>
            )}
          </div>
        ) : (
          <div className="space-y-3">
            {/* Workflow selector - always visible */}
            <p className="text-sm text-muted-foreground">
              Workflows provide agents with pre-defined context and structured steps to follow.
            </p>
            
            <div>
              <Popover open={popoverOpen} onOpenChange={(open) => {
                setPopoverOpen(open);
                if (open) {
                  setWorkflowSearch("");
                  // Focus the search input after a brief delay to ensure it's rendered
                  setTimeout(() => searchInputRef.current?.focus(), 0);
                }
              }}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={popoverOpen}
                    className="w-full h-auto py-3 justify-between"
                    disabled={workflowActivating}
                  >
                    {workflowActivating ? (
                      <div className="flex flex-col items-start gap-0.5 w-full">
                        <div className="flex items-center gap-2">
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          <span>Switching workflow...</span>
                        </div>
                        <span className="text-xs text-muted-foreground font-normal">
                          This may take a few seconds...
                        </span>
                      </div>
                    ) : (
                      <div className="flex items-start justify-between w-full gap-2">
                        <div className="flex flex-col items-start gap-0.5 text-left flex-1 min-w-0">
                          <span className="font-medium truncate w-full">{getSelectedWorkflowInfo().name}</span>
                          <span className="text-xs text-muted-foreground font-normal line-clamp-2 w-full">
                            {getSelectedWorkflowInfo().description}
                          </span>
                        </div>
                        <ChevronDown className="h-4 w-4 shrink-0 opacity-50 mt-1" />
                      </div>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[450px] p-0" align="start" sideOffset={4}>
                  {/* Search box */}
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
                        onKeyDown={(e) => {
                          // Prevent popover from closing on keyboard interaction
                          e.stopPropagation();
                        }}
                      />
                    </div>
                  </div>

                  {/* Workflow items */}
                  <div className="max-h-[400px] overflow-y-auto">
                    {showGeneralChat && (
                      <>
                        <button
                          onClick={() => handleWorkflowSelect("none")}
                          className={cn(
                            "w-full text-left px-2 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer",
                            selectedWorkflow === "none" && "bg-accent"
                          )}
                        >
                          <div className="flex flex-col items-start gap-0.5 py-1">
                            <span className="text-sm">General chat</span>
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
                        onClick={() => workflow.enabled && handleWorkflowSelect(workflow.id)}
                        disabled={!workflow.enabled}
                        className={cn(
                          "w-full text-left px-2 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer",
                          selectedWorkflow === workflow.id && "bg-accent",
                          !workflow.enabled && "opacity-50 cursor-not-allowed"
                        )}
                      >
                        <div className="flex flex-col items-start gap-0.5 py-1">
                          <span className="text-sm">{workflow.name}</span>
                          <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                            {workflow.description}
                          </span>
                        </div>
                      </button>
                    ))}
                    {(showGeneralChat || filteredWorkflows.length > 0) && showCustomWorkflow && (
                      <div className="border-t my-1" />
                    )}
                    {showCustomWorkflow && (
                      <button
                        onClick={() => handleWorkflowSelect("custom")}
                        className={cn(
                          "w-full text-left px-2 py-2 hover:bg-accent hover:text-accent-foreground cursor-pointer",
                          selectedWorkflow === "custom" && "bg-accent"
                        )}
                      >
                        <div className="flex flex-col items-start gap-0.5 py-1">
                          <span className="text-sm">Custom workflow...</span>
                          <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                            Load a workflow from a custom Git repository
                          </span>
                        </div>
                      </button>
                    )}
                    {!showGeneralChat && filteredWorkflows.length === 0 && !showCustomWorkflow && (
                      <div className="px-2 py-6 text-center text-sm text-muted-foreground">
                        No workflows found
                      </div>
                    )}
                  </div>
                </PopoverContent>
              </Popover>
            </div>
            
            {/* Show active workflow info */}
            {activeWorkflow && !workflowActivating && (
              <></>
            )}
          </div>
        )}
      </AccordionContent>
    </AccordionItem>
  );
}
