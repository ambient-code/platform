"use client";

import { Play, Loader2, Workflow, AlertCircle } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { WorkflowConfig } from "../../lib/types";

type WorkflowMetadata = {
  commands: Array<{ id: string; name: string; slashCommand: string; description?: string }>;
  agents: Array<{ id: string; name: string; description?: string }>;
};

type WorkflowsAccordionProps = {
  sessionPhase?: string;
  activeWorkflow: string | null;
  selectedWorkflow: string;
  pendingWorkflow: WorkflowConfig | null;
  workflowActivating: boolean;
  workflowMetadata?: WorkflowMetadata;
  ootbWorkflows: WorkflowConfig[];
  isExpanded: boolean;
  onWorkflowChange: (value: string) => void;
  onActivateWorkflow: () => void;
  onCommandClick: (slashCommand: string) => void;
  onResume?: () => void;
};

export function WorkflowsAccordion({
  sessionPhase,
  activeWorkflow,
  selectedWorkflow,
  pendingWorkflow,
  workflowActivating,
  workflowMetadata,
  ootbWorkflows,
  isExpanded,
  onWorkflowChange,
  onActivateWorkflow,
  onCommandClick,
  onResume,
}: WorkflowsAccordionProps) {
  const isSessionStopped = sessionPhase === 'Stopped' || sessionPhase === 'Error' || sessionPhase === 'Completed';

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
            {/* Workflow selector - always visible except when activating */}
            {!workflowActivating && (
              <>
                <p className="text-sm text-muted-foreground">
                  Workflows provide agents with pre-defined context and structured steps to follow.
                </p>
                
                <div>
                  <Select value={selectedWorkflow} onValueChange={onWorkflowChange} disabled={workflowActivating}>
                    <SelectTrigger className="w-full h-auto py-8">
                      <SelectValue placeholder="Generic chat" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">
                        <div className="flex flex-col items-start gap-0.5 py-1 max-w-[400px]">
                          <span>General chat</span>
                          <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                            A general chat session with no structured workflow.
                          </span>
                        </div>
                      </SelectItem>
                      {ootbWorkflows.map((workflow) => (
                        <SelectItem 
                          key={workflow.id} 
                          value={workflow.id}
                          disabled={!workflow.enabled}
                        >
                          <div className="flex flex-col items-start gap-0.5 py-1 max-w-[400px]">
                            <span>{workflow.name}</span>
                            <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                              {workflow.description}
                            </span>
                          </div>
                        </SelectItem>
                      ))}
                      <SelectSeparator />
                      <SelectItem value="custom">
                        <div className="flex flex-col items-start gap-0.5 py-1 max-w-[400px]">
                          <span>Custom workflow...</span>
                          <span className="text-xs text-muted-foreground font-normal line-clamp-2">
                            Load a workflow from a custom Git repository
                          </span>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                
                {/* Show workflow preview and activate/switch button */}
                {pendingWorkflow && (
                  <Alert variant="info">
                    <AlertCircle />
                    <AlertTitle>
                      Reload required
                    </AlertTitle>
                    <AlertDescription>
                      <div className="space-y-2 mt-2">
                        <p className="text-sm">
                          Please reload this chat session to switch to the new workflow. Your chat history will be preserved.
                        </p>
                        <Button 
                          onClick={onActivateWorkflow}
                          className="w-full mt-3"
                          size="sm"
                        >
                          <Play className="mr-2 h-4 w-4" />
                          Load new workflow
                        </Button>
                      </div>
                    </AlertDescription>
                  </Alert>
                )}
              </>
            )}
            
            {/* Show active workflow info */}
            {activeWorkflow && !workflowActivating && (
              <></>
            )}
            
            {/* Show activating/switching state */}
            {workflowActivating && (
              <Alert>
                <Loader2 className="h-4 w-4 animate-spin" />
                <AlertTitle>{activeWorkflow ? 'Switching Workflow...' : 'Activating Workflow...'}</AlertTitle>
                <AlertDescription>
                  <div className="space-y-2">
                    <p>Please wait. This may take 10-20 seconds...</p>
                  </div>
                </AlertDescription>
              </Alert>
            )}
          </div>
        )}
      </AccordionContent>
    </AccordionItem>
  );
}

