"use client";

import { useState, useCallback } from "react";
import { errorToast } from "@/hooks/use-toast";
import { useSessionQueue } from "@/hooks/use-session-queue";
import type { WorkflowConfig } from "../lib/types";

type UseWorkflowManagementProps = {
  projectName: string;
  sessionName: string;
  sessionPhase?: string;
  onWorkflowActivated?: () => void;
};

export function useWorkflowManagement({
  projectName,
  sessionName,
  sessionPhase,
  onWorkflowActivated,
}: UseWorkflowManagementProps) {
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>("none");
  const [pendingWorkflow, setPendingWorkflow] = useState<WorkflowConfig | null>(null);
  const [activeWorkflow, setActiveWorkflow] = useState<string | null>(null);
  const [workflowActivating, setWorkflowActivating] = useState(false);

  // Use session queue for workflow persistence
  const sessionQueue = useSessionQueue(projectName, sessionName);

  // Set pending workflow (user selected but not yet activated)
  const setPending = useCallback((workflow: WorkflowConfig | null) => {
    setPendingWorkflow(workflow);
  }, []);

  // Activate the pending workflow (or a workflow passed directly)
  const activateWorkflow = useCallback(async (workflowToActivate?: WorkflowConfig, currentPhase?: string, retryCount = 0) => {
    const workflow = workflowToActivate || pendingWorkflow;
    if (!workflow) return false;
    
    const phase = currentPhase || sessionPhase;
    
    // If session is not yet running, queue the workflow for later
    // This includes: undefined (loading), "Pending", "Creating", or any other non-Running state
    if (!phase || phase !== "Running") {
      sessionQueue.setWorkflow({
        id: workflow.id,
        gitUrl: workflow.gitUrl,
        branch: workflow.branch,
        path: workflow.path || "",
      });
      setSelectedWorkflow(workflow.id);
      setWorkflowActivating(true); // Show loading state
      return true; // Don't return false - we've queued it successfully
    }
    
    // Only set loading state on first attempt (not retries)
    if (retryCount === 0) {
      setWorkflowActivating(true);
    }
    
    try {
      // Update CR with workflow configuration
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
        
        // If runner not ready and we haven't retried too many times, retry with backoff
        if (errorData.retryable && retryCount < 5) {
          const delay = Math.min(1000 * Math.pow(1.5, retryCount), 5000); // Exponential backoff, max 5s
          console.log(`Runner not ready, retrying in ${delay}ms (attempt ${retryCount + 1}/5)...`);
          await new Promise(resolve => setTimeout(resolve, delay));
          // Retry without resetting loading state
          return activateWorkflow(workflow, phase, retryCount + 1);
        }
        
        throw new Error(errorData.error || "Failed to update workflow");
      }
      
      setActiveWorkflow(workflow.id);
      setPendingWorkflow(null);
      sessionQueue.clearWorkflow();
      
      // Wait for restart to complete (give runner time to clone and restart)
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      onWorkflowActivated?.();
      
      setWorkflowActivating(false);
      return true;
    } catch (error) {
      console.error("Failed to activate workflow:", error);
      errorToast(error instanceof Error ? error.message : "Failed to activate workflow");
      sessionQueue.clearWorkflow();
      setWorkflowActivating(false);
      return false;
    }
  }, [pendingWorkflow, projectName, sessionName, sessionPhase, sessionQueue, onWorkflowActivated]);

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
    queuedWorkflow: sessionQueue.workflow,
    activeWorkflow,
    setActiveWorkflow,
    workflowActivating,
    activateWorkflow,
    handleWorkflowChange,
    setCustomWorkflow,
  };
}

