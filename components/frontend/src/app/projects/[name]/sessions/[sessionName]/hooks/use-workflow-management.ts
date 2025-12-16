"use client";

import { useState, useCallback } from "react";
import { errorToast } from "@/hooks/use-toast";
import type { WorkflowConfig } from "../lib/types";

type UseWorkflowManagementProps = {
  projectName: string;
  sessionName: string;
  onWorkflowActivated?: () => void;
};

export function useWorkflowManagement({
  projectName,
  sessionName,
  onWorkflowActivated,
}: UseWorkflowManagementProps) {
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>("none");
  const [pendingWorkflow, setPendingWorkflow] = useState<WorkflowConfig | null>(null);
  const [activeWorkflow, setActiveWorkflow] = useState<string | null>(null);
  const [workflowActivating, setWorkflowActivating] = useState(false);

  // Set pending workflow (user selected but not yet activated)
  const setPending = useCallback((workflow: WorkflowConfig | null) => {
    setPendingWorkflow(workflow);
  }, []);

  // Activate the pending workflow (or a workflow passed directly)
  const activateWorkflow = useCallback(async (workflowToActivate?: WorkflowConfig) => {
    const workflow = workflowToActivate || pendingWorkflow;
    if (!workflow) return false;
    
    setWorkflowActivating(true);
    
    try {
      // 1. Update CR with workflow configuration
      const response = await fetch(`/api/projects/${projectName}/agentic-sessions/${sessionName}/workflow`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          gitUrl: workflow.gitUrl,
          branch: workflow.branch,
          path: workflow.path || "",
        }),
      });
      
      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to update workflow");
      }
      
      // 2. Send WebSocket message to trigger workflow clone and restart
      await fetch(`/api/projects/${projectName}/agentic-sessions/${sessionName}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: "workflow_change",
          gitUrl: workflow.gitUrl,
          branch: workflow.branch,
          path: workflow.path || "",
        }),
      });
      
      setActiveWorkflow(workflow.id);
      setPendingWorkflow(null);
      
      // Wait for restart to complete (give runner time to clone and restart)
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      onWorkflowActivated?.();
      
      return true;
    } catch (error) {
      console.error("Failed to activate workflow:", error);
      errorToast(error instanceof Error ? error.message : "Failed to activate workflow");
      return false;
    } finally {
      setWorkflowActivating(false);
    }
  }, [pendingWorkflow, projectName, sessionName, onWorkflowActivated]);

  // Handle workflow selection change
  const handleWorkflowChange = useCallback((value: string, ootbWorkflows: WorkflowConfig[], onCustom: () => void) => {
    setSelectedWorkflow(value);
    
    if (value === "none") {
      setPendingWorkflow(null);
      return null;
    }
    
    if (value === "custom") {
      onCustom();
      return null;
    }
    
    // Find the selected workflow from OOTB workflows
    const workflow = ootbWorkflows.find(w => w.id === value);
    if (!workflow) {
      errorToast(`Workflow ${value} not found`);
      return null;
    }
    
    if (!workflow.enabled) {
      errorToast(`Workflow ${workflow.name} is not yet available`);
      return null;
    }
    
    // Set as pending (user must click Activate)
    setPendingWorkflow(workflow);
    return workflow;
  }, []);

  // Set custom workflow as pending
  const setCustomWorkflow = useCallback((url: string, branch: string, path: string) => {
    setPendingWorkflow({
      id: "custom",
      name: "Custom workflow",
      description: `Custom workflow from ${url}`,
      gitUrl: url,
      branch: branch || "main",
      path: path || "",
      enabled: true,
    });
    setSelectedWorkflow("custom");
  }, []);

  return {
    selectedWorkflow,
    setSelectedWorkflow,
    pendingWorkflow,
    setPending,
    activeWorkflow,
    setActiveWorkflow,
    workflowActivating,
    activateWorkflow,
    handleWorkflowChange,
    setCustomWorkflow,
  };
}

