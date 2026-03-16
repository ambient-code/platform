/**
 * Workflow-related types shared across the application.
 */

/**
 * Configuration for an out-of-the-box (OOTB) workflow.
 * Represents a pre-defined workflow that can be activated in a session.
 */
export type WorkflowConfig = {
  id: string;
  name: string;
  description: string;
  gitUrl: string;
  branch: string;
  path?: string;
  enabled: boolean;
};

export type WorkflowCommand = {
  id: string;
  name: string;
  slashCommand: string;
  description?: string;
  icon?: string;
};

export type WorkflowAgent = {
  id: string;
  name: string;
  description?: string;
};

export type WorkflowMetadata = {
  commands: Array<WorkflowCommand>;
  agents: Array<WorkflowAgent>;
};

/**
 * Selection criteria for activating a workflow.
 * Used when creating or updating a session's active workflow.
 */
export type WorkflowSelection = {
  gitUrl: string;
  branch: string;
  path?: string;
};
